package toolgen_test

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/soyvural/mcp-server-openapi/toolgen"
)

func TestGenerate_Store(t *testing.T) {
	ctx := context.Background()
	tools, err := toolgen.Generate(ctx, toolgen.GenerateOptions{
		SpecSource: "../testdata/store.yaml",
		Tag:        "mcp",
	})
	if err != nil {
		t.Fatalf("TestGenerate_Store: got: %v, want: nil error", err)
	}

	if len(tools) != 3 {
		t.Fatalf("TestGenerate_Store: got: %d tools, want: 3", len(tools))
	}

	// Sort tools by name for deterministic assertions.
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name < tools[j].Name
	})

	wantNames := []string{"createitem", "getitembyid", "listitems"}
	var gotNames []string
	for _, tool := range tools {
		gotNames = append(gotNames, tool.Name)
	}
	if diff := cmp.Diff(wantNames, gotNames); diff != "" {
		t.Errorf("TestGenerate_Store: tool names mismatch (-want, +got): %v", diff)
	}

	// Verify deleteItem (internal tag) is excluded.
	for _, tool := range tools {
		if tool.Name == "deleteitem" {
			t.Errorf("TestGenerate_Store: got: deleteitem tool present, want: excluded (internal tag)")
		}
	}

	// Verify server URL propagated to all tools.
	for _, tool := range tools {
		if tool.ServerURL != "https://store.example.com/v1" {
			t.Errorf("TestGenerate_Store: got: ServerURL=%q for %q, want: https://store.example.com/v1", tool.ServerURL, tool.Name)
		}
	}
}

func TestGenerate_Extensions(t *testing.T) {
	ctx := context.Background()
	tools, err := toolgen.Generate(ctx, toolgen.GenerateOptions{
		SpecSource: "../testdata/extensions.yaml",
		Tag:        "mcp",
	})
	if err != nil {
		t.Fatalf("TestGenerate_Extensions: got: %v, want: nil error", err)
	}

	if len(tools) != 2 {
		t.Fatalf("TestGenerate_Extensions: got: %d tools, want: 2", len(tools))
	}

	// Sort tools by name for deterministic assertions.
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name < tools[j].Name
	})

	wantNames := []string{"custom_visible", "forcedop"}
	var gotNames []string
	for _, tool := range tools {
		gotNames = append(gotNames, tool.Name)
	}
	if diff := cmp.Diff(wantNames, gotNames); diff != "" {
		t.Errorf("TestGenerate_Extensions: tool names mismatch (-want, +got): %v", diff)
	}

	// Verify custom_visible has custom description from x-mcp-description.
	for _, tool := range tools {
		if tool.Name == "custom_visible" {
			want := "Custom description here"
			if tool.Description != want {
				t.Errorf("TestGenerate_Extensions: got: Description=%q for custom_visible, want: %q", tool.Description, want)
			}
		}
	}

	// Verify hiddenOp is excluded (x-mcp-hidden: true).
	for _, tool := range tools {
		if tool.Name == "hiddenop" {
			t.Errorf("TestGenerate_Extensions: got: hiddenop tool present, want: excluded (x-mcp-hidden: true)")
		}
	}
}

func TestGenerate_Collision(t *testing.T) {
	ctx := context.Background()
	_, err := toolgen.Generate(ctx, toolgen.GenerateOptions{
		SpecSource: "../testdata/collision.yaml",
		Tag:        "mcp",
	})
	if err == nil {
		t.Fatalf("TestGenerate_Collision: got: nil error, want: collision error")
	}
	if !strings.Contains(err.Error(), "collision") {
		t.Errorf("TestGenerate_Collision: got: %v, want: error containing 'collision'", err)
	}
}
