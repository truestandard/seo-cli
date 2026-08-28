package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	"github.com/truestandard/seo-cli/internal/api"
	"github.com/truestandard/seo-cli/internal/output"
)

const mcpProtocolVersion = "2025-06-18"

var projectArgumentNames = []string{"project", "slug", "project_slug"}

type backendTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func mcpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "stdio MCP server that proxies the backend's tools (Claude Code, Codex, Cursor)",
		Long: "Starts a Model Context Protocol server on stdin/stdout. Register it with " +
			"`claude mcp add seo -- seo mcp` or the equivalent block in .cursor/mcp.json or ~/.codex/config.toml. " +
			"It fetches the tool list from the backend at startup and fills in the default project.",
		Args: exactArgs(0, "seo mcp"),
		RunE: func(_ *cobra.Command, _ []string) error {
			client, err := requireClient()
			if err != nil {
				return err
			}
			tools, err := handshake(client)
			if err != nil {
				return err
			}
			server := newProxyServer(client, tools, cfg.Project)
			output.Notice("seo mcp: proxying %d tools from %s", len(tools), client.BaseURL)
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()
			return server.Run(ctx, &mcp.StdioTransport{})
		},
	}
}

func handshake(client *api.Client) ([]backendTool, error) {
	if _, err := client.McpRequest("initialize", map[string]any{
		"protocolVersion": mcpProtocolVersion,
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "seo-cli", "version": version},
	}); err != nil {
		return nil, err
	}
	if err := client.McpNotify("notifications/initialized", map[string]any{}); err != nil {
		return nil, err
	}
	listed, err := client.McpRequest("tools/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	var result struct {
		Tools []backendTool `json:"tools"`
	}
	if err := json.Unmarshal(listed, &result); err != nil {
		return nil, &api.APIError{Code: "mcp_invalid_response", Message: "tools/list returned an unexpected shape"}
	}
	if len(result.Tools) == 0 {
		return nil, &api.APIError{Code: "mcp_no_tools", Message: fmt.Sprintf("backend at %s returned no MCP tools", client.BaseURL)}
	}
	return result.Tools, nil
}

func newProxyServer(client *api.Client, tools []backendTool, project string) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "seo", Title: "TrueStandard Agency", Version: version}, nil)
	for _, tool := range tools {
		tool := tool
		server.AddTool(&mcp.Tool{Name: tool.Name, Description: tool.Description, InputSchema: tool.InputSchema},
			func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				args := WithDefaultProject(tool.InputSchema, req.Params.Arguments, project)
				return proxyCall(client, tool.Name, args)
			})
	}
	return server
}

func WithDefaultProject(schema map[string]any, raw json.RawMessage, project string) map[string]any {
	args := map[string]any{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &args)
	}
	if project == "" {
		return args
	}
	properties, _ := schema["properties"].(map[string]any)
	for _, name := range projectArgumentNames {
		if _, declared := properties[name]; declared {
			if _, given := args[name]; !given {
				args[name] = project
			}
		}
	}
	return args
}

func proxyCall(client *api.Client, name string, args map[string]any) (*mcp.CallToolResult, error) {
	raw, err := client.McpRequest("tools/call", map[string]any{"name": name, "arguments": args})
	if err != nil {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}, IsError: true}, nil
	}
	var result mcp.CallToolResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(raw)}}}, nil
	}
	return &result, nil
}
