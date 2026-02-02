package toolhandler

type ToolRequest struct {
	SessionId string         `json:"session_id"`
	Arguments map[string]any `json:"arguments"`
}
