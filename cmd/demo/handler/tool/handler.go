package tool

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

type toolHandler struct{}

func (h *toolHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	w.Header().Set("Content-Type", "application/json")

	// Discovery: Empty body returns the tool list
	if len(raw) == 0 {
		response := map[string]any{
			"version": "1.0",
			"tools":   tools,
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Execution
	var args map[string]any
	if err := json.Unmarshal(raw, &args); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if _, ok := args["message"]; ok {
		log.Printf("🛠️ Executing 'echo'")
		json.NewEncoder(w).Encode(map[string]any{"result": args["message"]})
		return
	} else if _, ok := args["format"]; ok {
		log.Printf("🛠️ Executing 'timestamp'")
		json.NewEncoder(w).Encode(map[string]any{"result": time.Now().Format(time.RFC3339)})
		return
	}

	http.Error(w, "unknown tool signature", http.StatusBadRequest)
}

func NewHandler() *toolHandler {
	return &toolHandler{}
}
