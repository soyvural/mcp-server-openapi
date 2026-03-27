// Package server wires tools and executor into an MCP server.
package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/soyvural/mcp-server-openapi/executor"
	"github.com/soyvural/mcp-server-openapi/toolgen"

	mcpserver "github.com/mark3labs/mcp-go/server"
)

// New creates an MCP server with the given tools.
func New(tools []toolgen.GeneratedTool, exec executor.RequestExecutor, version string) (*mcpserver.MCPServer, error) {
	if len(tools) == 0 {
		return nil, fmt.Errorf("no tools generated from spec — check your tag filter")
	}

	s := mcpserver.NewMCPServer(
		"mcp-server-openapi",
		version,
	)

	for _, gt := range tools {
		tool := mcp.NewToolWithRawSchema(gt.Name, gt.Description, gt.InputSchema)
		handler := makeHandler(gt, exec)
		s.AddTool(tool, handler)
		slog.Debug("registered tool", "name", gt.Name, "method", gt.Method, "path", gt.Path)
	}

	slog.Info("MCP server ready", "tools", len(tools))
	return s, nil
}

func makeHandler(gt toolgen.GeneratedTool, exec executor.RequestExecutor) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		toolReq := &executor.ToolRequest{
			ServerURL: gt.ServerURL,
			Path:      gt.Path,
			Method:    gt.Method,
			Args:      args,
			ParamMeta: gt.ParamMeta,
		}

		resp, err := exec.Execute(ctx, toolReq)
		if err != nil {
			return nil, fmt.Errorf("tool %q: %w", gt.Name, err)
		}

		if resp.IsError {
			msg := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Body)
			return mcp.NewToolResultError(msg), nil
		}

		return mcp.NewToolResultText(resp.Body), nil
	}
}
