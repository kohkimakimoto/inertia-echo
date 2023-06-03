// package ssr is currently under development...

package ssr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/kohkimakimoto/inertia-echo"
	"net/http"
	"strings"
)

type NodeSsrEngine struct {
	SsrURL     string
	HttpClient *http.Client
}

func NewNodeSsrEngine() *NodeSsrEngine {
	return &NodeSsrEngine{
		SsrURL:     "http://127.0.0.1:13714",
		HttpClient: &http.Client{},
	}
}

func (e *NodeSsrEngine) Render(p *inertia.Page) (*inertia.SsrResponse, error) {
	pJson, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(
		http.MethodPost,
		strings.ReplaceAll(e.SsrURL, "/render", "")+"/render",
		bytes.NewBuffer(pJson),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ssr: status code is not 200: %d", resp.StatusCode)
	}

	var ssrResp inertia.SsrResponse
	err = json.NewDecoder(resp.Body).Decode(&ssrResp)
	if err != nil {
		return nil, err
	}
	return &ssrResp, nil
}
