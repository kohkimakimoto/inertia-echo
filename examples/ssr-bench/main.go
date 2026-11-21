package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"syscall"
	"time"

	inertia "github.com/kohkimakimoto/inertia-echo/v5"
)

type benchRow struct {
	name       string
	init       time.Duration
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
	flag.Parse()

	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	bundle := *bundleFlag
	if bundle == "" {
		bundle = defaultSlowjsBundle(root)
	}

	if *engineFlag == "all" && *isolated {
		fmt.Printf("# SSR comparison: slowjs (in-process) vs Node (HTTP SSR)\n\n")
		fmt.Printf("- slowjs bundle: `%s`\n", bundle)
		fmt.Printf("- node: `examples/ssr-node` → `node .build/ssr/app.js` @ http://127.0.0.1:13714\n")
		fmt.Printf("- page: Index / %d warmup + %d timed renders\n", *warmup, *iters)
		fmt.Printf("- go: %s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
		fmt.Printf("- note: architectures differ on purpose (thin IIFE vs official Node SSR path)\n\n")

		var rows []benchRow
		for _, name := range []string{"slowjs", "node"} {
			r, err := runIsolated(name, bundle, *warmup, *iters)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", name, err)
				os.Exit(1)
			}
			rows = append(rows, r)
		}
		printPerf(rows)
		printRecommendation(rows)
		return
	}

	eng := engineFor(*engineFlag, root, bundle)
	row := measure(eng, *warmup, *iters)
	if *engineFlag != "all" {
		printIsolatedJSON(row)
		return
	}
	printPerf([]benchRow{row})
}

func engineFor(name, root, bundle string) Engine {
	switch name {
	case "slowjs":
		return newSlowjsEngine(root, bundle)
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

func measure(eng Engine, warmup, iters int) benchRow {
	r := benchRow{name: eng.Name()}
	page := samplePage()

	runtime.GC()
	t0 := time.Now()
	if err := eng.Open(); err != nil {
		r.openErr = err
		eng.Close()
		return r
	}
	r.init = time.Since(t0)

	for i := 0; i < warmup; i++ {
		if _, err := eng.Render(page); err != nil {
			r.renderErr = fmt.Errorf("warmup: %w", err)
			eng.Close()
			return r
		}
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

	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	r.heapInuse = ms.HeapInuse
	r.selfRSS = selfMaxRSS()
	r.extraRSS = eng.ExtraRSSBytes()
	eng.Close()
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
	InitNs     int64    `json:"init_ns"`
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
		InitNs:     int64(r.init),
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

func runIsolated(engine, bundle string, warmup, iters int) (benchRow, error) {
	args := []string{"run", ".",
		"-engine", engine,
		"-isolated=false",
		"-bundle", bundle,
		"-warmup", fmt.Sprintf("%d", warmup),
		"-n", fmt.Sprintf("%d", iters),
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
		init:       time.Duration(p.InitNs),
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

func printPerf(rows []benchRow) {
	fmt.Printf("## Render latency & memory\n\n")
	fmt.Printf("| Engine | init | Go HeapInuse | Go MaxRSS | Node RSS | footprint≈ | n | avg | p50 | p95 | p99 | min | max | body_len |\n")
	fmt.Printf("|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, r := range rows {
		if r.openErr != nil {
			fmt.Printf("| %s | — | — | — | — | — | — | OPEN: %v |\n", r.name, r.openErr)
			continue
		}
		if r.renderErr != nil {
			fmt.Printf("| %s | %s | %s | %s | %s | — | — | RENDER: %v |\n",
				r.name, r.init.Round(time.Millisecond), humanBytes(r.heapInuse), humanBytes(r.selfRSS), humanBytes(r.extraRSS), r.renderErr)
			continue
		}
		s := summarize(r.latencies)
		foot := r.selfRSS
		if r.extraRSS > 0 {
			foot = r.selfRSS + r.extraRSS
		}
		fmt.Printf("| %s | %s | %s | %s | %s | %s | %d | %s | %s | %s | %s | %s | %s | %d |\n",
			r.name,
			r.init.Round(time.Millisecond),
			humanBytes(r.heapInuse),
			humanBytes(r.selfRSS),
			humanBytes(r.extraRSS),
			humanBytes(foot),
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
	fmt.Printf("\n- **Go MaxRSS**: peak RSS via `Getrusage` (includes VM for slowjs; harness-only for node).\n")
	fmt.Printf("- **Node RSS**: `process.memoryUsage().rss` via preload `:13715/__mem`.\n")
	fmt.Printf("- **footprint≈**: slowjs = Go MaxRSS; node = Go MaxRSS + Node RSS.\n")
	fmt.Printf("- Node latency includes localhost HTTP to the SSR server.\n\n")
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

func printRecommendation(rows []benchRow) {
	fmt.Printf("## Adoption notes (from this run)\n\n")
	var slow, node *benchRow
	for i := range rows {
		switch rows[i].name {
		case "slowjs":
			slow = &rows[i]
		case "node":
			node = &rows[i]
		}
	}
	if slow == nil || node == nil || slow.openErr != nil || node.openErr != nil || slow.renderErr != nil || node.renderErr != nil {
		fmt.Printf("Could not compare both engines successfully.\n")
		return
	}
	slowAvg := summarize(slow.latencies).avg
	nodeAvg := summarize(node.latencies).avg
	fmt.Printf("- **Latency (avg)**: slowjs=%s, node=%s (%.2fx).\n",
		slowAvg.Round(time.Microsecond), nodeAvg.Round(time.Microsecond), float64(nodeAvg)/float64(slowAvg))
	fmt.Printf("- **Memory**: slowjs MaxRSS=%s; node MaxRSS+NodeRSS=%s+%s.\n",
		humanBytes(slow.selfRSS), humanBytes(node.selfRSS), humanBytes(node.extraRSS))
	fmt.Printf("- **Deploy shape**: slowjs = one Go binary; node = Go + Node SSR process.\n\n")
	fmt.Printf("**Recommendation**: For inertia-echo React SSR, prefer **slowjs** when you want in-process SSR without a Node sidecar; keep **ssr-node** when you want the official Inertia/Node SSR path (full Node API surface, Vite `--ssr`).\n")
}
