package utcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	goutcp "github.com/universal-tool-calling-protocol/go-utcp"
	toolhandler "github.com/w-h-a/agent/tool_handler"
	"github.com/w-h-a/agent/tool_handler/utcp"
	toolprovider "github.com/w-h-a/agent/tool_provider"
)

type utcpToolProvider struct {
	options toolprovider.Options
	client  goutcp.UtcpClientInterface
}

func (tp *utcpToolProvider) Load(ctx context.Context, query string, limit int) ([]toolhandler.ToolHandler, error) {
	remoteTools, err := tp.client.SearchTools(query, limit)
	if err != nil {
		return nil, fmt.Errorf("utcp discovery failed: %w", err)
	}

	var handlers []toolhandler.ToolHandler
	for _, tool := range remoteTools {
		spec := toolhandler.ToolSpec{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.Inputs.Properties,
		}
		handlers = append(handlers, utcp.NewToolHandler(
			utcp.WithUtcpClient(tp.client),
			utcp.WithToolName(tool.Name),
			utcp.WithToolSpec(spec),
		))
	}

	return handlers, nil
}

func (tp *utcpToolProvider) createTempConfig(addrs []string) (string, error) {
	type providerConfig struct {
		Type    string            `json:"provider_type"`
		Name    string            `json:"name"`
		URL     string            `json:"url"`
		Method  string            `json:"http_method"`
		Headers map[string]string `json:"headers"`
	}

	config := struct {
		Providers []providerConfig `json:"providers"`
	}{}

	for _, u := range addrs {
		parsed, err := url.Parse(u)
		if err != nil {
			return "", err
		}
		name := parsed.Hostname()
		config.Providers = append(config.Providers, providerConfig{
			Type:   "http",
			Name:   name,
			URL:    u,
			Method: "POST",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		})
	}

	f, err := os.CreateTemp("", "utcp_config_*.json")
	if err != nil {
		return "", err
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(config); err != nil {
		return "", err
	}

	return f.Name(), nil
}

func NewToolProvider(opts ...toolprovider.Option) toolprovider.ToolProvider {
	options := toolprovider.NewOptions(opts...)

	tp := &utcpToolProvider{
		options: options,
	}

	var configPath string

	if len(options.Addrs) > 0 {
		tmpPath, err := tp.createTempConfig(options.Addrs)
		if err != nil {
			panic(err)
		}
		configPath = tmpPath
		defer os.Remove(tmpPath)
	}

	client, err := goutcp.NewUTCPClient(
		context.Background(),
		&goutcp.UtcpClientConfig{
			ProvidersFilePath: configPath,
		},
		nil,
		nil,
	)
	if err != nil {
		panic(err)
	}

	tp.client = client

	return tp
}
