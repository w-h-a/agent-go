package tool

var tools = []map[string]any{
	{
		"name":        "echo",
		"description": "Echo back a message",
		"inputs": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"message": map[string]string{"type": "string"},
			},
			"required": []string{"message"},
		},
	},
	{
		"name":        "timestamp",
		"description": "Current server timestamp",
		"inputs": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"format": map[string]string{
					"type":        "string",
					"description": "Time format (e.g., RFC3339)",
				},
			},
			"required": []string{"format"},
		},
	},
}
