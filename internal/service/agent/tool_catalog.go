package agent

import (
	"fmt"
	"strings"
	"sync"

	toolhandler "github.com/w-h-a/agent/tool_handler"
)

type ToolCatalog struct {
	tools map[string]toolhandler.ToolHandler
	specs map[string]toolhandler.ToolSpec
	order []string
	mtx   sync.RWMutex
}

func (c *ToolCatalog) Register(th toolhandler.ToolHandler) error {
	if th == nil {
		return fmt.Errorf("tool is nil")
	}

	c.mtx.Lock()
	defer c.mtx.Unlock()

	spec := th.Spec()
	key := strings.ToLower(strings.TrimSpace(spec.Name))
	if len(key) == 0 {
		return fmt.Errorf("tool name is required")
	}

	if _, ok := c.tools[key]; ok {
		return fmt.Errorf("tool %s already registered", key)
	}

	c.tools[key] = th
	c.specs[key] = spec
	c.order = append(c.order, key)

	return nil
}

func (c *ToolCatalog) ListSpecs() []toolhandler.ToolSpec {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	specs := make([]toolhandler.ToolSpec, 0, len(c.specs))
	for _, key := range c.order {
		specs = append(specs, c.specs[key])
	}

	return specs
}

func (c *ToolCatalog) Get(name string) (toolhandler.ToolHandler, toolhandler.ToolSpec, bool) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	key := strings.ToLower(strings.TrimSpace(name))
	tp, ok := c.tools[key]

	return tp, c.specs[key], ok
}
