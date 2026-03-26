package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/soyvural/mcp-server-openapi/executor"
	"github.com/soyvural/mcp-server-openapi/server"
	"github.com/soyvural/mcp-server-openapi/toolgen"
)

func TestE2E_ToolCallRoundTrip(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/tasks":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "title": "Buy milk", "done": false}})
		case r.Method == "POST" && r.URL.Path == "/tasks":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 2, "title": "New task", "done": false})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer upstream.Close()

	tools := []toolgen.GeneratedTool{
		{
			Name:        "list_tasks",
			Description: "List all tasks",
			InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
			Path:        "/tasks",
			Method:      "GET",
			ServerURL:   upstream.URL,
		},
		{
			Name:        "create_task",
			Description: "Create a task",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"title":{"type":"string"}},"required":["title"]}`),
			ParamMeta:   []toolgen.ParamMeta{{Name: "title", In: "body"}},
			Path:        "/tasks",
			Method:      "POST",
			ServerURL:   upstream.URL,
		},
	}

	exec := executor.New(upstream.Client(), &executor.NoAuth{}, 10*time.Second)
	mcpServer, err := server.New(tools, exec, "test")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Retrieve tool handlers via GetTool and call them directly.
	ctx := context.Background()

	t.Run("list_tasks", func(t *testing.T) {
		st := mcpServer.GetTool("list_tasks")
		if st == nil {
			t.Fatal("GetTool returned nil for list_tasks")
		}
		result, err := st.Handler(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "list_tasks",
				Arguments: map[string]any{},
			},
		})
		if err != nil {
			t.Fatalf("list_tasks call failed: %v", err)
		}
		if len(result.Content) == 0 {
			t.Fatal("empty result content")
		}
		tc, ok := result.Content[0].(mcp.TextContent)
		if !ok {
			t.Fatalf("expected TextContent, got %T", result.Content[0])
		}
		if !json.Valid([]byte(tc.Text)) {
			t.Errorf("response not valid JSON: %s", tc.Text)
		}
		t.Logf("list_tasks response: %s", tc.Text)
	})

	t.Run("create_task", func(t *testing.T) {
		st := mcpServer.GetTool("create_task")
		if st == nil {
			t.Fatal("GetTool returned nil for create_task")
		}
		result, err := st.Handler(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "create_task",
				Arguments: map[string]any{"title": "New task"},
			},
		})
		if err != nil {
			t.Fatalf("create_task call failed: %v", err)
		}
		if len(result.Content) == 0 {
			t.Fatal("empty result content")
		}
		tc, ok := result.Content[0].(mcp.TextContent)
		if !ok {
			t.Fatalf("expected TextContent, got %T", result.Content[0])
		}
		if !json.Valid([]byte(tc.Text)) {
			t.Errorf("response not valid JSON: %s", tc.Text)
		}
		t.Logf("create_task response: %s", tc.Text)
	})
}

func TestE2E_ToolCallNotFound(t *testing.T) {
	tools := []toolgen.GeneratedTool{
		{
			Name:        "dummy",
			Description: "A dummy tool",
			InputSchema: json.RawMessage(`{"type":"object"}`),
			Path:        "/dummy",
			Method:      "GET",
			ServerURL:   "http://localhost:0",
		},
	}

	mcpServer, err := server.New(tools, nil, "test")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	st := mcpServer.GetTool("nonexistent")
	if st != nil {
		t.Errorf("expected nil for nonexistent tool, got %v", st)
	}
}

func TestE2E_UpstreamError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal server error"}`))
	}))
	defer upstream.Close()

	tools := []toolgen.GeneratedTool{
		{
			Name:        "failing_tool",
			Description: "A tool that hits a failing upstream",
			InputSchema: json.RawMessage(`{"type":"object"}`),
			Path:        "/fail",
			Method:      "GET",
			ServerURL:   upstream.URL,
		},
	}

	exec := executor.New(upstream.Client(), &executor.NoAuth{}, 10*time.Second)
	mcpServer, err := server.New(tools, exec, "test")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	st := mcpServer.GetTool("failing_tool")
	if st == nil {
		t.Fatal("GetTool returned nil for failing_tool")
	}

	result, err := st.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "failing_tool",
			Arguments: map[string]any{},
		},
	})
	if err != nil {
		t.Fatalf("handler returned unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError to be true for 500 response")
	}
	if len(result.Content) == 0 {
		t.Fatal("empty result content")
	}
	tc, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if tc.Text == "" {
		t.Error("expected non-empty error text")
	}
	t.Logf("error response: %s", tc.Text)
}
