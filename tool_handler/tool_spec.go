package toolhandler

type ToolSpec struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	InputSchema map[string]any   `json:"input_schema"`
	Examples    []map[string]any `json:"examples,omitempty"`
}
