package toolprovider

type ToolRequest struct {
	SessionId string         `json:"session_id"`
	Arguments map[string]any `json:"arguments"`
}
