package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	inertia "github.com/kohkimakimoto/inertia-echo/v5"
)

type benchRow struct {
	name       string
	mode       string // latency | throughput
	poolSize   int
	workers    int
	init       time.Duration
	wall       time.Duration
	heapInuse  uint64
	selfRSS    uint64
	extraRSS   uint64
	latencies  []time.Duration
	bodyLen    int
	headSample []string
	openErr    error
	renderErr  error
}

func main() {
	iters := flag.Int("n", 200, "render iterations after warmup")
	warmup := flag.Int("warmup", 20, "warmup renders (not timed)")
	engineFlag := flag.String("engine", "all", "slowjs | node | all")
	isolated := flag.Bool("isolated", true, "when -engine=all, run each engine in a fresh process")
	bundleFlag := flag.String("bundle", "", "slowjs SSR IIFE path (default: ../ssr-slowjs/.build/ssr/ssr.js)")
	modeFlag := flag.String("mode", "latency", "latency | throughput | both")
	// -1 = pick by mode (latency→1, throughput→0/GOMAXPROCS). 0 = library default GOMAXPROCS.
	poolFlag := flag.Int("pool", -1, "ssr/slowjs PoolSize (-1=mode default, 0=GOMAXPROCS, N=fixed)")
	workersFlag := flag.Int("workers", 0, "throughput concurrent workers (0 = GOMAXPROCS)")
	flag.Parse()

	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	bundle := *bundleFlag
	if bundle == "" {
		bundle = defaultSlowjsBundle(root)
	}

	modes, err := parseModes(*modeFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(2)
	}
	workers := resolveWorkers(*workersFlag)

	if *engineFlag == "all" && *isolated {
		fmt.Printf("# SSR comparison: slowjs (`ssr/slowjs`) vs Node (`ssr/nodejs`)\n\n")
		fmt.Printf("- slowjs bundle: `%s`\n", bundle)
		fmt.Printf("- node: `examples/ssr-node` → `node .build/ssr/app.js` via `ssr/nodejs.Gateway`\n")
		fmt.Printf("- page: Index / %d warmup + %d timed renders\n", *warmup, *iters)
		fmt.Printf("- go: %s %s/%s GOMAXPROCS=%d\n", runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.GOMAXPROCS(0))
		fmt.Printf("- note: architectures differ on purpose (thin IIFE vs official Node SSR path)\n\n")

		var latencyRows, throughputRows []benchRow
		for _, mode := range modes {
			pool := resolvePool(mode, *poolFlag)
			if mode == "latency" {
				fmt.Printf("## Mode: latency (sequential Dispatch, pool default 1)\n\n")
			} else {
				fmt.Printf("## Mode: throughput (%d concurrent workers, slowjs pool default GOMAXPROCS)\n\n", workers)
			}
			var rows []benchRow
			for _, name := range []string{"slowjs", "node"} {
				r, err := runIsolated(name, bundle, mode, *warmup, *iters, pool, workers)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%s/%s: %v\n", mode, name, err)
					os.Exit(1)
				}
				rows = append(rows, r)
			}
			if mode == "latency" {
				printLatency(rows)
				latencyRows = rows
			} else {
				printThroughput(rows)
				throughputRows = rows
			}
		}
		printRecommendation(latencyRows, throughputRows)
		return
	}

	if len(modes) != 1 {
		fmt.Fprintf(os.Stderr, "child process requires a single -mode (latency|throughput)\n")
		os.Exit(2)
	}
	mode := modes[0]
	pool := resolvePool(mode, *poolFlag)
	eng := engineFor(*engineFlag, root, bundle, pool)
	var row benchRow
	if mode == "throughput" {
		row = measureThroughput(eng, *warmup, *iters, workers)
	} else {
		row = measureLatency(eng, *warmup, *iters)
	}
	if *engineFlag != "all" {
		printIsolatedJSON(row)
		return
	}
	if mode == "throughput" {
		printThroughput([]benchRow{row})
	} else {
		printLatency([]benchRow{row})
	}
}

func parseModes(s string) ([]string, error) {
	switch s {
	case "latency":
		return []string{"latency"}, nil
	case "throughput":
		return []string{"throughput"}, nil
	case "both":
		return []string{"latency", "throughput"}, nil
	default:
		return nil, fmt.Errorf("unknown -mode %q (want latency|throughput|both)", s)
	}
}

func resolvePool(mode string, poolFlag int) int {
	if poolFlag >= 0 {
		return poolFlag
	}
	if mode == "throughput" {
		return 0 // library default: GOMAXPROCS
	}
	return 1
}

func resolveWorkers(n int) int {
	if n > 0 {
		return n
	}
	w := runtime.GOMAXPROCS(0)
	if w < 1 {
		return 1
	}
	return w
}

func engineFor(name, root, bundle string, poolSize int) Engine {
	switch name {
	case "slowjs":
		return newSlowjsEngine(root, bundle, poolSize)
	case "node":
		return newNodeEngine(root)
	default:
		fmt.Fprintf(os.Stderr, "unknown -engine %q\n", name)
		os.Exit(2)
		return nil
	}
}

func samplePage() *inertia.Page {
	return &inertia.Page{
		Component: "Index",
		Props: map[string]any{
			"title":   "bench",
			"message": "same props, both engines",
		},
		URL:     "/",
		Version: "bench",
	}
}

func openEngine(eng Engine, r *benchRow) bool {
	runtime.GC()
	t0 := time.Now()
	if err := eng.Open(); err != nil {
		r.openErr = err
		eng.Close()
		return false
	}
	r.init = time.Since(t0)
	if s, ok := eng.(*slowjsEngine); ok && s.gateway != nil {
		r.poolSize = s.gateway.PoolSize()
	}
	return true
}

func warmupEngine(eng Engine, page *inertia.Page, warmup int, r *benchRow) bool {
	for i := 0; i < warmup; i++ {
		if _, err := eng.Render(page); err != nil {
			r.renderErr = fmt.Errorf("warmup: %w", err)
			eng.Close()
			return false
		}
	}
	return true
}

func finishMem(eng Engine, r *benchRow) {
	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	r.heapInuse = ms.HeapInuse
	r.selfRSS = selfMaxRSS()
	r.extraRSS = eng.ExtraRSSBytes()
	eng.Close()
}

func measureLatency(eng Engine, warmup, iters int) benchRow {
	r := benchRow{name: eng.Name(), mode: "latency"}
	page := samplePage()
	if !openEngine(eng, &r) {
		return r
	}
	if !warmupEngine(eng, page, warmup, &r) {
		return r
	}

	r.latencies = make([]time.Duration, 0, iters)
	for i := 0; i < iters; i++ {
		t1 := time.Now()
		res, err := eng.Render(page)
		dt := time.Since(t1)
		if err != nil {
			r.renderErr = err
			eng.Close()
			return r
		}
		r.bodyLen = len(res.Body)
		r.headSample = res.Head
		r.latencies = append(r.latencies, dt)
	}
	r.wall = 0
	for _, d := range r.latencies {
		r.wall += d
	}
	finishMem(eng, &r)
	return r
}

func measureThroughput(eng Engine, warmup, iters, workers int) benchRow {
	r := benchRow{name: eng.Name(), mode: "throughput", workers: workers}
	page := samplePage()
	if !openEngine(eng, &r) {
		return r
	}
	if !warmupEngine(eng, page, warmup, &r) {
		return r
	}
	if workers < 1 {
		workers = 1
		r.workers = 1
	}

	lat := make([]time.Duration, iters)
	var fail atomic.Value // error
	var metaOnce sync.Once
	var bodyLen int
	var headSample []string
	var wg sync.WaitGroup
	jobs := make(chan int, iters)
	for i := 0; i < iters; i++ {
		jobs <- i
	}
	close(jobs)

	wallStart := time.Now()
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				if fail.Load() != nil {
					return
				}
				t1 := time.Now()
				res, err := eng.Render(page)
				dt := time.Since(t1)
				if err != nil {
					fail.Store(err)
					return
				}
				lat[i] = dt
				metaOnce.Do(func() {
					bodyLen = len(res.Body)
					headSample = res.Head
				})
			}
		}()
	}
	wg.Wait()
	r.wall = time.Since(wallStart)

	if err, ok := fail.Load().(error); ok && err != nil {
		r.renderErr = err
		eng.Close()
		return r
	}
	r.bodyLen = bodyLen
	r.headSample = headSample
	r.latencies = lat
	finishMem(eng, &r)
	return r
}

// selfMaxRSS is peak RSS for this process (darwin: bytes; linux: KiB in older Go — normalize via GOOS).
func selfMaxRSS() uint64 {
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		return 0
	}
	rss := uint64(ru.Maxrss)
	if runtime.GOOS == "linux" {
		rss *= 1024
	}
	return rss
}

type isolatedPayload struct {
	Name       string   `json:"name"`
	Mode       string   `json:"mode"`
	PoolSize   int      `json:"pool_size,omitempty"`
	Workers    int      `json:"workers,omitempty"`
	InitNs     int64    `json:"init_ns"`
	WallNs     int64    `json:"wall_ns"`
	HeapInuse  uint64   `json:"heap_inuse"`
	SelfRSS    uint64   `json:"self_rss"`
	ExtraRSS   uint64   `json:"extra_rss"`
	BodyLen    int      `json:"body_len"`
	HeadSample []string `json:"head_sample"`
	LatNs      []int64  `json:"lat_ns"`
	OpenErr    string   `json:"open_err,omitempty"`
	RenderErr  string   `json:"render_err,omitempty"`
}

func printIsolatedJSON(r benchRow) {
	p := isolatedPayload{
		Name:       r.name,
		Mode:       r.mode,
		PoolSize:   r.poolSize,
		Workers:    r.workers,
		InitNs:     int64(r.init),
		WallNs:     int64(r.wall),
		HeapInuse:  r.heapInuse,
		SelfRSS:    r.selfRSS,
		ExtraRSS:   r.extraRSS,
		BodyLen:    r.bodyLen,
		HeadSample: r.headSample,
	}
	if r.openErr != nil {
		p.OpenErr = r.openErr.Error()
	}
	if r.renderErr != nil {
		p.RenderErr = r.renderErr.Error()
	}
	for _, d := range r.latencies {
		p.LatNs = append(p.LatNs, int64(d))
	}
	_ = json.NewEncoder(os.Stdout).Encode(p)
}

func runIsolated(engine, bundle, mode string, warmup, iters, poolSize, workers int) (benchRow, error) {
	args := []string{"run", ".",
		"-engine", engine,
		"-isolated=false",
		"-mode", mode,
		"-bundle", bundle,
		"-warmup", fmt.Sprintf("%d", warmup),
		"-n", fmt.Sprintf("%d", iters),
		"-pool", fmt.Sprintf("%d", poolSize),
		"-workers", fmt.Sprintf("%d", workers),
	}
	cmd := exec.Command("go", args...)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return benchRow{}, fmt.Errorf("%v\n%s", err, ee.Stderr)
		}
		return benchRow{}, err
	}
	var p isolatedPayload
	if err := json.Unmarshal(trimSpaceBytes(out), &p); err != nil {
		return benchRow{}, fmt.Errorf("decode child: %w\n%s", err, out)
	}
	r := benchRow{
		name:       p.Name,
		mode:       p.Mode,
		poolSize:   p.PoolSize,
		workers:    p.Workers,
		init:       time.Duration(p.InitNs),
		wall:       time.Duration(p.WallNs),
		heapInuse:  p.HeapInuse,
		selfRSS:    p.SelfRSS,
		extraRSS:   p.ExtraRSS,
		bodyLen:    p.BodyLen,
		headSample: p.HeadSample,
	}
	if p.OpenErr != "" {
		r.openErr = fmt.Errorf("%s", p.OpenErr)
	}
	if p.RenderErr != "" {
		r.renderErr = fmt.Errorf("%s", p.RenderErr)
	}
	for _, ns := range p.LatNs {
		r.latencies = append(r.latencies, time.Duration(ns))
	}
	return r, nil
}

func trimSpaceBytes(b []byte) []byte {
	i, j := 0, len(b)
	for i < j && (b[i] == ' ' || b[i] == '\n' || b[i] == '\r' || b[i] == '\t') {
		i++
	}
	for j > i && (b[j-1] == ' ' || b[j-1] == '\n' || b[j-1] == '\r' || b[j-1] == '\t') {
		j--
	}
	return b[i:j]
}

func poolLabel(r benchRow) string {
	if r.name == "slowjs" && r.poolSize > 0 {
		return fmt.Sprintf("%d", r.poolSize)
	}
	return "—"
}

func footprint(r benchRow) uint64 {
	if r.extraRSS > 0 {
		return r.selfRSS + r.extraRSS
	}
	return r.selfRSS
}

func rps(r benchRow) float64 {
	if r.wall <= 0 || len(r.latencies) == 0 {
		return 0
	}
	return float64(len(r.latencies)) / r.wall.Seconds()
}

func printLatency(rows []benchRow) {
	fmt.Printf("### Render latency & memory (sequential)\n\n")
	fmt.Printf("| Engine | pool | init | Go HeapInuse | Go MaxRSS | Node RSS | footprint≈ | n | avg | p50 | p95 | p99 | min | max | body_len |\n")
	fmt.Printf("|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, r := range rows {
		pool := poolLabel(r)
		if r.openErr != nil {
			fmt.Printf("| %s | %s | — | — | — | — | — | — | OPEN: %v |\n", r.name, pool, r.openErr)
			continue
		}
		if r.renderErr != nil {
			fmt.Printf("| %s | %s | %s | %s | %s | %s | — | — | RENDER: %v |\n",
				r.name, pool, r.init.Round(time.Millisecond), humanBytes(r.heapInuse), humanBytes(r.selfRSS), humanBytes(r.extraRSS), r.renderErr)
			continue
		}
		s := summarize(r.latencies)
		fmt.Printf("| %s | %s | %s | %s | %s | %s | %s | %d | %s | %s | %s | %s | %s | %s | %d |\n",
			r.name,
			pool,
			r.init.Round(time.Millisecond),
			humanBytes(r.heapInuse),
			humanBytes(r.selfRSS),
			humanBytes(r.extraRSS),
			humanBytes(footprint(r)),
			len(r.latencies),
			s.avg.Round(time.Microsecond),
			s.p50.Round(time.Microsecond),
			s.p95.Round(time.Microsecond),
			s.p99.Round(time.Microsecond),
			s.min.Round(time.Microsecond),
			s.max.Round(time.Microsecond),
			r.bodyLen,
		)
	}
	fmt.Printf("\n- **Go MaxRSS**: peak RSS via `Getrusage` (includes VM pool for slowjs; harness-only for node).\n")
	fmt.Printf("- **Node RSS**: `process.memoryUsage().rss` via preload `:13715/__mem`.\n")
	fmt.Printf("- **footprint≈**: slowjs = Go MaxRSS; node = Go MaxRSS + Node RSS.\n")
	fmt.Printf("- Node latency includes localhost HTTP to the SSR server.\n\n")
}

func printThroughput(rows []benchRow) {
	fmt.Printf("### Throughput (concurrent)\n\n")
	fmt.Printf("| Engine | pool | workers | init | footprint≈ | n | wall | rps | avg | p50 | p95 | p99 |\n")
	fmt.Printf("|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, r := range rows {
		pool := poolLabel(r)
		if r.openErr != nil {
			fmt.Printf("| %s | %s | %d | — | — | — | — | — | OPEN: %v |\n", r.name, pool, r.workers, r.openErr)
			continue
		}
		if r.renderErr != nil {
			fmt.Printf("| %s | %s | %d | %s | %s | — | — | — | RENDER: %v |\n",
				r.name, pool, r.workers, r.init.Round(time.Millisecond), humanBytes(footprint(r)), r.renderErr)
			continue
		}
		s := summarize(r.latencies)
		fmt.Printf("| %s | %s | %d | %s | %s | %d | %s | %.0f | %s | %s | %s | %s |\n",
			r.name,
			pool,
			r.workers,
			r.init.Round(time.Millisecond),
			humanBytes(footprint(r)),
			len(r.latencies),
			r.wall.Round(time.Millisecond),
			rps(r),
			s.avg.Round(time.Microsecond),
			s.p50.Round(time.Microsecond),
			s.p95.Round(time.Microsecond),
			s.p99.Round(time.Microsecond),
		)
	}
	fmt.Printf("\n- **rps**: timed renders / wall-clock (parallel workers).\n")
	fmt.Printf("- **avg/p50/…**: per-render latency under concurrent load.\n")
	fmt.Printf("- slowjs pool defaults to GOMAXPROCS in this mode; Node is one HTTP SSR process.\n\n")
}

type stats struct {
	avg, p50, p95, p99, min, max time.Duration
}

func summarize(ds []time.Duration) stats {
	if len(ds) == 0 {
		return stats{}
	}
	cp := append([]time.Duration(nil), ds...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	var sum time.Duration
	for _, d := range cp {
		sum += d
	}
	pct := func(p float64) time.Duration {
		i := int(float64(len(cp)-1) * p)
		return cp[i]
	}
	return stats{
		avg: sum / time.Duration(len(cp)),
		p50: pct(0.50),
		p95: pct(0.95),
		p99: pct(0.99),
		min: cp[0],
		max: cp[len(cp)-1],
	}
}

func humanBytes(n uint64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := uint64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func printRecommendation(latency, throughput []benchRow) {
	fmt.Printf("## Adoption notes (from this run)\n\n")
	slowL, nodeL := findEngines(latency)
	if slowL != nil && nodeL != nil && slowL.openErr == nil && nodeL.openErr == nil && slowL.renderErr == nil && nodeL.renderErr == nil {
		slowAvg := summarize(slowL.latencies).avg
		nodeAvg := summarize(nodeL.latencies).avg
		fmt.Printf("- **Latency (avg, sequential)**: slowjs=%s, node=%s (%.2fx).\n",
			slowAvg.Round(time.Microsecond), nodeAvg.Round(time.Microsecond), float64(nodeAvg)/float64(slowAvg))
		fmt.Printf("- **Memory (latency run)**: slowjs MaxRSS=%s; node MaxRSS+NodeRSS=%s+%s.\n",
			humanBytes(slowL.selfRSS), humanBytes(nodeL.selfRSS), humanBytes(nodeL.extraRSS))
	}
	slowT, nodeT := findEngines(throughput)
	if slowT != nil && nodeT != nil && slowT.openErr == nil && nodeT.openErr == nil && slowT.renderErr == nil && nodeT.renderErr == nil {
		fmt.Printf("- **Throughput**: slowjs=%.0f rps (pool=%d, workers=%d), node=%.0f rps (workers=%d) (%.2fx).\n",
			rps(*slowT), slowT.poolSize, slowT.workers, rps(*nodeT), nodeT.workers, rps(*slowT)/rps(*nodeT))
		fmt.Printf("- **Memory (throughput run)**: slowjs MaxRSS=%s; node footprint≈%s.\n",
			humanBytes(slowT.selfRSS), humanBytes(footprint(*nodeT)))
	}
	fmt.Printf("- **Deploy shape**: slowjs = one Go binary; node = Go + Node SSR process.\n\n")
	fmt.Printf("**Recommendation**: For inertia-echo React SSR, prefer **slowjs** when you want in-process SSR without a Node sidecar; keep **ssr-node** when you want the official Inertia/Node SSR path (full Node API surface, Vite `--ssr`).\n")
}

func findEngines(rows []benchRow) (slow, node *benchRow) {
	for i := range rows {
		switch rows[i].name {
		case "slowjs":
			slow = &rows[i]
		case "node":
			node = &rows[i]
		}
	}
	return slow, node
}
