package agent

import (
	"fmt"
	"strings"
	"sync"

	toolprovider "github.com/w-h-a/agent/tool_provider"
)

type Catalog struct {
	tools map[string]toolprovider.ToolProvider
	specs map[string]toolprovider.ToolSpec
	order []string
	mtx   sync.RWMutex
}

func (c *Catalog) Register(tp toolprovider.ToolProvider) error {
	if tp == nil {
		return fmt.Errorf("tool is nil")
	}

	c.mtx.Lock()
	defer c.mtx.Unlock()

	spec := tp.Spec()
	key := strings.ToLower(strings.TrimSpace(spec.Name))
	if len(key) == 0 {
		return fmt.Errorf("tool name is required")
	}

	if _, ok := c.tools[key]; ok {
		return fmt.Errorf("tool %s already registered", key)
	}

	c.tools[key] = tp
	c.specs[key] = spec
	c.order = append(c.order, key)

	return nil
}

func (c *Catalog) ListSpecs() []toolprovider.ToolSpec {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	specs := make([]toolprovider.ToolSpec, 0, len(c.specs))
	for _, key := range c.order {
		specs = append(specs, c.specs[key])
	}

	return specs
}

func (c *Catalog) Get(name string) (toolprovider.ToolProvider, toolprovider.ToolSpec, bool) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	key := strings.ToLower(strings.TrimSpace(name))
	tp, ok := c.tools[key]

	return tp, c.specs[key], ok
}
