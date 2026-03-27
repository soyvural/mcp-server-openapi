package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/soyvural/mcp-server-openapi/toolgen"
)

func TestExecute_GET_WithPathAndQuery(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"status":"ok"}`)
	}))
	defer srv.Close()

	exec := New(srv.Client(), &NoAuth{}, 5*time.Second)
	resp, err := exec.Execute(context.Background(), &ToolRequest{
		ServerURL: srv.URL,
		Path:      "/items/{id}/toys",
		Method:    http.MethodGet,
		Args: map[string]any{
			"id": "42",
			"limit": 10,
		},
		ParamMeta: []toolgen.ParamMeta{
			{Name: "id", In: "path"},
			{Name: "limit", In: "query"},
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if gotPath != "/items/42/toys" {
		t.Errorf("path: got: %q, want: %q", gotPath, "/items/42/toys")
	}
	if gotQuery != "limit=10" {
		t.Errorf("query: got: %q, want: %q", gotQuery, "limit=10")
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got: %d, want: %d", resp.StatusCode, http.StatusOK)
	}
	if resp.Body != `{"status":"ok"}` {
		t.Errorf("body: got: %q, want: %q", resp.Body, `{"status":"ok"}`)
	}
	if resp.IsError {
		t.Errorf("IsError: got: true, want: false")
	}
}

func TestExecute_POST_WithBody(t *testing.T) {
	var gotContentType string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprint(w, `{"id":1}`)
	}))
	defer srv.Close()

	exec := New(srv.Client(), &NoAuth{}, 5*time.Second)
	resp, err := exec.Execute(context.Background(), &ToolRequest{
		ServerURL: srv.URL,
		Path:      "/items",
		Method:    http.MethodPost,
		Args: map[string]any{
			"name": "Widget",
			"tag":  "electronics",
		},
		ParamMeta: []toolgen.ParamMeta{
			{Name: "name", In: "body"},
			{Name: "tag", In: "body"},
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if gotContentType != "application/json" {
		t.Errorf("Content-Type: got: %q, want: %q", gotContentType, "application/json")
	}
	if gotBody["name"] != "Widget" {
		t.Errorf("body.name: got: %v, want: %q", gotBody["name"], "Widget")
	}
	if gotBody["tag"] != "electronics" {
		t.Errorf("body.tag: got: %v, want: %q", gotBody["tag"], "electronics")
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status: got: %d, want: %d", resp.StatusCode, http.StatusCreated)
	}
	if resp.Body != `{"id":1}` {
		t.Errorf("body: got: %q, want: %q", resp.Body, `{"id":1}`)
	}
	if resp.IsError {
		t.Errorf("IsError: got: true, want: false")
	}
}

func TestExecute_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"error":"not found"}`)
	}))
	defer srv.Close()

	exec := New(srv.Client(), &NoAuth{}, 5*time.Second)
	resp, err := exec.Execute(context.Background(), &ToolRequest{
		ServerURL: srv.URL,
		Path:      "/items/{id}",
		Method:    http.MethodGet,
		Args: map[string]any{
			"id": "999",
		},
		ParamMeta: []toolgen.ParamMeta{
			{Name: "id", In: "path"},
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status: got: %d, want: %d", resp.StatusCode, http.StatusNotFound)
	}
	if resp.Body != `{"error":"not found"}` {
		t.Errorf("body: got: %q, want: %q", resp.Body, `{"error":"not found"}`)
	}
	if !resp.IsError {
		t.Errorf("IsError: got: false, want: true")
	}
}
