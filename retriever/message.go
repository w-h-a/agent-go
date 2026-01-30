package retriever

type Message struct {
	Id        string    `json:"id"`
	SessionId string    `json:"session_id"`
	Role      string    `json:"role"`
	Parts     []Part    `json:"parts"`
	Embedding []float32 `json:"embedding"`
}

type Part struct {
	Type      string         `json:"type"`
	Text      string         `json:"text,omitempty"`
	FileField string         `json:"file_field,omitempty"`
	Meta      map[string]any `json:"meta,omitempty"`
}
