package toolgen_test

import (
	"context"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/soyvural/mcp-server-openapi/toolgen"
)

func TestLoadSpec(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		desc    string
		source  string
		wantErr bool
	}{
		{
			desc:    "load from file success",
			source:  "../testdata/store.yaml",
			wantErr: false,
		},
		{
			desc:    "file not found error",
			source:  "../testdata/nonexistent.yaml",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			doc, err := toolgen.LoadSpec(ctx, tc.source)
			if tc.wantErr {
				if err == nil {
					t.Errorf("%s: got: nil error, want: error", tc.desc)
				}
				return
			}
			if err != nil {
				t.Errorf("%s: got: %v, want: nil error", tc.desc, err)
				return
			}
			if doc == nil {
				t.Errorf("%s: got: nil doc, want: non-nil doc", tc.desc)
			}
		})
	}
}

func TestLoadSpecPathCount(t *testing.T) {
	ctx := context.Background()

	doc, err := toolgen.LoadSpec(ctx, "../testdata/store.yaml")
	if err != nil {
		t.Fatalf("failed to load store spec: %v", err)
	}

	got := len(doc.Paths.Map())
	want := 2
	if got != want {
		t.Errorf("TestLoadSpecPathCount: got: %d paths, want: %d paths", got, want)
	}
}

func TestServerURL(t *testing.T) {
	ctx := context.Background()

	doc, err := toolgen.LoadSpec(ctx, "../testdata/store.yaml")
	if err != nil {
		t.Fatalf("failed to load store spec: %v", err)
	}

	tests := []struct {
		desc     string
		doc      *openapi3.T
		override string
		want     string
		wantErr  bool
	}{
		{
			desc:     "with override",
			doc:      doc,
			override: "https://override.example.com/",
			want:     "https://override.example.com",
			wantErr:  false,
		},
		{
			desc:     "from spec",
			doc:      doc,
			override: "",
			want:     "https://store.example.com/v1",
			wantErr:  false,
		},
		{
			desc:     "no server error",
			doc:      &openapi3.T{},
			override: "",
			want:     "",
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := toolgen.ServerURL(tc.doc, tc.override)
			if tc.wantErr {
				if err == nil {
					t.Errorf("%s: got: nil error, want: error", tc.desc)
				}
				return
			}
			if err != nil {
				t.Errorf("%s: got: %v, want: nil error", tc.desc, err)
				return
			}
			if got != tc.want {
				t.Errorf("%s: got: %v, want: %v", tc.desc, got, tc.want)
			}
		})
	}
}
