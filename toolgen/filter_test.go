package toolgen_test

import (
	"testing"

	"github.com/soyvural/mcp-server-openapi/toolgen"
)

func TestTagFilterInclude(t *testing.T) {
	tests := []struct {
		desc    string
		tag     string
		opTags  []string
		opExt   map[string]any
		pathExt map[string]any
		want    bool
	}{
		{
			desc:    "include: has matching tag",
			tag:     "users",
			opTags:  []string{"users", "admin"},
			opExt:   map[string]any{},
			pathExt: map[string]any{},
			want:    true,
		},
		{
			desc:    "exclude: no matching tag",
			tag:     "users",
			opTags:  []string{"admin", "posts"},
			opExt:   map[string]any{},
			pathExt: map[string]any{},
			want:    false,
		},
		{
			desc:   "exclude: has tag but x-mcp-hidden=true",
			tag:    "users",
			opTags: []string{"users"},
			opExt: map[string]any{
				"x-mcp-hidden": true,
			},
			pathExt: map[string]any{},
			want:    false,
		},
		{
			desc:   "include: no tag but x-mcp-hidden=false",
			tag:    "users",
			opTags: []string{"admin"},
			opExt: map[string]any{
				"x-mcp-hidden": false,
			},
			pathExt: map[string]any{},
			want:    true,
		},
		{
			desc:   "exclude: path hidden overrides tag",
			tag:    "users",
			opTags: []string{"users"},
			opExt:  map[string]any{},
			pathExt: map[string]any{
				"x-mcp-hidden": true,
			},
			want: false,
		},
		{
			desc:   "include: op hidden=false overrides path hidden=true",
			tag:    "users",
			opTags: []string{"admin"},
			opExt: map[string]any{
				"x-mcp-hidden": false,
			},
			pathExt: map[string]any{
				"x-mcp-hidden": true,
			},
			want: true,
		},
		{
			desc:    "exclude: empty tags no extensions",
			tag:     "users",
			opTags:  []string{},
			opExt:   map[string]any{},
			pathExt: map[string]any{},
			want:    false,
		},
		{
			desc:   "include: hidden=false with empty tags",
			tag:    "users",
			opTags: []string{},
			opExt: map[string]any{
				"x-mcp-hidden": false,
			},
			pathExt: map[string]any{},
			want:    true,
		},
		{
			desc:   "exclude: hidden=true overrides matching tag",
			tag:    "users",
			opTags: []string{"users", "admin"},
			opExt:  map[string]any{},
			pathExt: map[string]any{
				"x-mcp-hidden": true,
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			filter := toolgen.NewTagFilter(tc.tag)
			got := filter.Include(tc.opTags, tc.opExt, tc.pathExt)
			if got != tc.want {
				t.Errorf("%s: got: %v, want: %v", tc.desc, got, tc.want)
			}
		})
	}
}

func TestNewTagFilter(t *testing.T) {
	filter := toolgen.NewTagFilter("test-tag")
	if filter == nil {
		t.Errorf("NewTagFilter returned nil")
	}
}
