package memorymanager

import (
	"encoding/json"
)

type Task struct {
	Id        string          `json:"id"`
	SessionId string          `json:"session_id"`
	TaskOrder int             `json:"task_order"`
	Data      json.RawMessage `json:"data"`
	Status    string          `json:"status"`
}
