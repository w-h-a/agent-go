package munin

import memorymanager "github.com/w-h-a/agent/memory_manager"

type sessionBuffer struct {
	spaceId  string
	messages []memorymanager.Message
}
