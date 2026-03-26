package server

import (
	"encoding/json"
	"testing"

	"github.com/soyvural/mcp-server-openapi/toolgen"
)

func TestNewServer_RegistersTools(t *testing.T) {
	tools := []toolgen.GeneratedTool{
		{
			Name:        "list_pets",
			Description: "List all pets",
			InputSchema: json.RawMessage(`{"type":"object"}`),
			Path:        "/pets",
			Method:      "GET",
			ServerURL:   "https://api.example.com",
		},
		{
			Name:        "create_pet",
			Description: "Create a pet",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}}}`),
			ParamMeta: []toolgen.ParamMeta{
				{Name: "name", In: "body", Required: true},
			},
			Path:      "/pets",
			Method:    "POST",
			ServerURL: "https://api.example.com",
		},
	}

	srv, err := New(tools, nil, "0.1.0")
	if err != nil {
		t.Fatalf("New() returned unexpected error: %v", err)
	}
	if srv == nil {
		t.Fatal("New() returned nil server")
	}
}

func TestNewServer_NoTools(t *testing.T) {
	_, err := New(nil, nil, "0.1.0")
	if err == nil {
		t.Fatal("New(nil, ...) expected error, got nil")
	}
}
