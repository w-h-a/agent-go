package toolprovider

type ToolResponse struct {
	Content  string            `json:"content"`
	Metadata map[string]string `json:"metadata,omitempty"`
}
