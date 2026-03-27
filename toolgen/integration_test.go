package toolgen_test

import (
	"context"
	"testing"

	"github.com/soyvural/mcp-server-openapi/toolgen"
)

func TestIntegration_StoreToolGeneration(t *testing.T) {
	tools, err := toolgen.Generate(context.Background(), toolgen.GenerateOptions{
		SpecSource: "../testdata/store.yaml",
		Tag:        "mcp",
	})
	if err != nil {
		t.Fatalf("failed to generate: %v", err)
	}
	if len(tools) != 3 {
		t.Fatalf("tool count: got %d, want 3", len(tools))
	}
	for _, tool := range tools {
		if tool.Name == "" {
			t.Error("tool has empty name")
		}
		if tool.Description == "" {
			t.Errorf("tool %q has empty description", tool.Name)
		}
		if len(tool.InputSchema) == 0 {
			t.Errorf("tool %q has empty input schema", tool.Name)
		}
		if tool.Path == "" {
			t.Errorf("tool %q has empty path", tool.Name)
		}
		if tool.Method == "" {
			t.Errorf("tool %q has empty method", tool.Name)
		}
		if tool.ServerURL == "" {
			t.Errorf("tool %q has empty server URL", tool.Name)
		}
	}
}

func TestIntegration_ExtensionsSpec(t *testing.T) {
	tools, err := toolgen.Generate(context.Background(), toolgen.GenerateOptions{
		SpecSource: "../testdata/extensions.yaml",
		Tag:        "mcp",
	})
	if err != nil {
		t.Fatalf("failed to generate: %v", err)
	}
	nameMap := make(map[string]toolgen.GeneratedTool)
	for _, tool := range tools {
		nameMap[tool.Name] = tool
	}
	if cv, ok := nameMap["custom_visible"]; !ok {
		t.Error("missing tool 'custom_visible'")
	} else if cv.Description != "Custom description here" {
		t.Errorf("custom_visible desc: got %q, want %q", cv.Description, "Custom description here")
	}
	if _, ok := nameMap["forcedop"]; !ok {
		t.Error("missing tool 'forcedop'")
	}
}

func TestIntegration_CollisionDetected(t *testing.T) {
	_, err := toolgen.Generate(context.Background(), toolgen.GenerateOptions{
		SpecSource: "../testdata/collision.yaml",
		Tag:        "mcp",
	})
	if err == nil {
		t.Fatal("expected collision error, got nil")
	}
}

func TestIntegration_DemoAPISpec(t *testing.T) {
	tools, err := toolgen.Generate(context.Background(), toolgen.GenerateOptions{
		SpecSource: "../examples/demo-api/openapi.yaml",
		Tag:        "mcp",
	})
	if err != nil {
		t.Fatalf("failed to generate: %v", err)
	}
	if len(tools) != 5 {
		t.Errorf("tool count: got %d, want 5", len(tools))
	}
	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name] = true
	}
	for _, want := range []string{"list_tasks", "create_task", "get_task", "update_task", "delete_task"} {
		if !names[want] {
			t.Errorf("missing expected tool %q", want)
		}
	}
}

func TestValidation_NoDuplicateToolNames(t *testing.T) {
	specs := []string{
		"../testdata/store.yaml",
		"../testdata/extensions.yaml",
		"../examples/demo-api/openapi.yaml",
	}
	for _, spec := range specs {
		t.Run(spec, func(t *testing.T) {
			tools, err := toolgen.Generate(context.Background(), toolgen.GenerateOptions{
				SpecSource: spec,
				Tag:        "mcp",
			})
			if err != nil {
				t.Skipf("spec failed: %v", err)
			}
			seen := make(map[string]bool)
			for _, tool := range tools {
				if seen[tool.Name] {
					t.Errorf("duplicate tool name: %q", tool.Name)
				}
				seen[tool.Name] = true
			}
		})
	}
}

func TestValidation_AllToolNamesAreValid(t *testing.T) {
	tools, err := toolgen.Generate(context.Background(), toolgen.GenerateOptions{
		SpecSource: "../testdata/store.yaml",
		Tag:        "mcp",
	})
	if err != nil {
		t.Fatalf("failed to generate: %v", err)
	}
	for _, tool := range tools {
		for _, c := range tool.Name {
			valid := (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '.' || c == '-'
			if !valid {
				t.Errorf("tool %q has invalid char %q", tool.Name, string(c))
			}
		}
	}
}
