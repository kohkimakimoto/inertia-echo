package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	inertia "github.com/kohkimakimoto/inertia-echo/v5"
)

type nodeEngine struct {
	root    string
	ssrDir  string
	url     string
	cmd     *exec.Cmd
	gateway *inertia.SsrHTTPGateway
}

func newNodeEngine(root string) *nodeEngine {
	return &nodeEngine{
		root:   root,
		ssrDir: filepath.Join(root, "..", "ssr-node"),
		url:    "http://127.0.0.1:13714",
	}
}

func (e *nodeEngine) Name() string { return "node" }

func (e *nodeEngine) Open() error {
	if _, err := os.Stat(filepath.Join(e.ssrDir, "package.json")); err != nil {
		return fmt.Errorf("ssr-node not found at %s: %w", e.ssrDir, err)
	}
	if _, err := os.Stat(filepath.Join(e.ssrDir, "node_modules")); err != nil {
		install := exec.Command("npm", "install")
		install.Dir = e.ssrDir
		install.Stdout = os.Stderr
		install.Stderr = os.Stderr
		if err := install.Run(); err != nil {
			return fmt.Errorf("npm install in ssr-node: %w", err)
		}
	}

	build := exec.Command("npm", "run", "build:ssr")
	build.Dir = e.ssrDir
	build.Stdout = os.Stderr
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("npm run build:ssr: %w", err)
	}

	bundle := filepath.Join(e.ssrDir, ".build", "ssr", "app.js")
	if _, err := os.Stat(bundle); err != nil {
		return fmt.Errorf("SSR bundle missing: %s", bundle)
	}

	preload, err := filepath.Abs(filepath.Join(e.root, "node_mem_preload.cjs"))
	if err != nil {
		return err
	}
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "node", "--require", preload, bundle)
	cmd.Dir = e.ssrDir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start node SSR: %w", err)
	}
	e.cmd = cmd

	e.gateway = &inertia.SsrHTTPGateway{
		Enabled:    true,
		URL:        e.url,
		HttpClient: &http.Client{Timeout: 10 * time.Second},
	}

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if e.cmd.ProcessState != nil && e.cmd.ProcessState.Exited() {
			return fmt.Errorf("node SSR exited early")
		}
		if e.gateway.IsHealthy(context.Background()) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	e.Close()
	return fmt.Errorf("node SSR health check timed out at %s/health", e.url)
}

func (e *nodeEngine) Close() {
	if e.cmd != nil && e.cmd.Process != nil {
		_ = e.cmd.Process.Kill()
		_, _ = e.cmd.Process.Wait()
		e.cmd = nil
	}
}

func (e *nodeEngine) Render(page *inertia.Page) (*inertia.SsrResponse, error) {
	return e.gateway.Dispatch(&inertia.RenderContext{Page: page})
}

func (e *nodeEngine) ExtraRSSBytes() uint64 {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:13715/__mem", nil)
	if err != nil {
		return 0
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0
	}
	defer res.Body.Close()
	var m struct {
		RSS uint64 `json:"rss"`
	}
	if err := json.NewDecoder(res.Body).Decode(&m); err != nil {
		return 0
	}
	return m.RSS
}
