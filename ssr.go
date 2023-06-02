package inertia

type SsrResponse struct {
	Head []string `json:"head"`
	Body string   `json:"body"`
}

type SsrEngine interface {
	Render(*Page) (*SsrResponse, error)
}
