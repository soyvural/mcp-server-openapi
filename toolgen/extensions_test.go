package toolgen_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/soyvural/mcp-server-openapi/toolgen"
)

func TestExtractExtensions(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := []struct {
		desc    string
		opExt   map[string]any
		pathExt map[string]any
		want    toolgen.MCPExtensions
	}{
		{
			desc:    "no extensions returns empty struct",
			opExt:   map[string]any{},
			pathExt: map[string]any{},
			want:    toolgen.MCPExtensions{},
		},
		{
			desc: "tool name from op",
			opExt: map[string]any{
				"x-mcp-tool-name": "custom_tool",
			},
			pathExt: map[string]any{},
			want: toolgen.MCPExtensions{
				ToolName: "custom_tool",
			},
		},
		{
			desc: "description from op",
			opExt: map[string]any{
				"x-mcp-description": "custom description",
			},
			pathExt: map[string]any{},
			want: toolgen.MCPExtensions{
				Description: "custom description",
			},
		},
		{
			desc: "hidden at op level overrides path",
			opExt: map[string]any{
				"x-mcp-hidden": true,
			},
			pathExt: map[string]any{
				"x-mcp-hidden": false,
			},
			want: toolgen.MCPExtensions{
				Hidden: &trueVal,
			},
		},
		{
			desc:  "hidden at path level when no op override",
			opExt: map[string]any{},
			pathExt: map[string]any{
				"x-mcp-hidden": false,
			},
			want: toolgen.MCPExtensions{
				Hidden: &falseVal,
			},
		},
		{
			desc: "all extensions from op",
			opExt: map[string]any{
				"x-mcp-tool-name":   "my_tool",
				"x-mcp-description": "my description",
				"x-mcp-hidden":      false,
			},
			pathExt: map[string]any{},
			want: toolgen.MCPExtensions{
				ToolName:    "my_tool",
				Description: "my description",
				Hidden:      &falseVal,
			},
		},
		{
			desc: "non-string values are ignored",
			opExt: map[string]any{
				"x-mcp-tool-name":   123,
				"x-mcp-description": true,
				"x-mcp-hidden":      "not-a-bool",
			},
			pathExt: map[string]any{},
			want:    toolgen.MCPExtensions{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			got := toolgen.ExtractExtensions(tc.opExt, tc.pathExt)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("%s: extensions mismatch (-want, +got):\n%s", tc.desc, diff)
			}
		})
	}
}
