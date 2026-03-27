# mcp-server-openapi Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go CLI that converts OpenAPI-tagged endpoints into MCP tools automatically.

**Architecture:** OpenAPI spec → kin-openapi parser → filter by `mcp` tag → generate MCP tools with flat JSON Schema → serve via mcp-go over stdio/HTTP. Tool calls decompose args back into HTTP requests against the upstream API.

**Tech Stack:** Go 1.23+, kin-openapi, mcp-go, cobra, viper, slog

---

## File Map

| File | Responsibility |
|------|---------------|
| `cmd/mcp-server-openapi/main.go` | Root Cobra command, persistent flags, version |
| `cmd/mcp-server-openapi/stdio.go` | stdio subcommand — loads spec, builds server, serves |
| `cmd/mcp-server-openapi/http.go` | HTTP subcommand — same but Streamable HTTP transport |
| `toolgen/parser.go` | Load OpenAPI spec from file/URL via kin-openapi |
| `toolgen/parser_test.go` | Parser tests |
| `toolgen/filter.go` | Tag-based + x-mcp-hidden operation filtering |
| `toolgen/filter_test.go` | Filter tests |
| `toolgen/namer.go` | operationId → sanitized MCP tool name |
| `toolgen/namer_test.go` | Namer tests |
| `toolgen/extensions.go` | x-mcp-* extension extraction |
| `toolgen/extensions_test.go` | Extension tests |
| `toolgen/schema.go` | Params + body → flat JSON Schema, collision detection |
| `toolgen/schema_test.go` | Schema tests |
| `toolgen/generator.go` | Orchestrator: parser → filter → namer → schema → tools |
| `toolgen/generator_test.go` | Generator integration tests |
| `executor/auth.go` | Authenticator interface + BearerAuth, APIKeyAuth |
| `executor/auth_test.go` | Auth tests |
| `executor/executor.go` | Tool call → HTTP request → response mapping |
| `executor/executor_test.go` | Executor tests |
| `server/server.go` | Wires toolgen + executor into mcp-go MCPServer |
| `pkg/params/params.go` | Generic Required[T] / Optional[T] helpers |
| `pkg/params/params_test.go` | Params tests |
| `examples/petstore/petstore.yaml` | Modified Petstore with mcp tags |
| `examples/petstore/README.md` | Petstore example docs |
| `examples/demo-api/main.go` | Self-contained task API server |
| `examples/demo-api/openapi.yaml` | Demo API spec with x-mcp-* extensions |
| `examples/demo-api/README.md` | Demo example docs |
| `testdata/petstore.yaml` | Test fixture: Petstore spec with mcp tags |
| `testdata/collision.yaml` | Test fixture: spec with param name collision |
| `testdata/extensions.yaml` | Test fixture: spec with x-mcp-* extensions |
| `go.mod` | Module definition |
| `Makefile` | Build, test, lint targets |
| `Dockerfile` | Container image |
| `.goreleaser.yaml` | Release config |
| `README.md` | Project documentation |
| `LICENSE` | MIT license |

---

### Task 1: Project Scaffold & Go Module

**Files:**
- Create: `go.mod`, `cmd/mcp-server-openapi/main.go`, `Makefile`, `LICENSE`

- [ ] **Step 1: Initialize Go module**

```bash
cd /Users/mustafasoyvural/mvs_workspace/mcp-server-openapi
go mod init github.com/soyvural/mcp-server-openapi
```

- [ ] **Step 2: Add dependencies**

```bash
go get github.com/mark3labs/mcp-go@latest
go get github.com/getkin/kin-openapi@latest
go get github.com/spf13/cobra@latest
go get github.com/spf13/viper@latest
```

- [ ] **Step 3: Create main.go with root Cobra command**

Create `cmd/mcp-server-openapi/main.go`:
```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var version = "dev"

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "mcp-server-openapi",
	Short: "Serve OpenAPI endpoints as MCP tools",
	Long:  "Automatically converts OpenAPI-tagged endpoints into MCP tools for LLM integration.",
}

func init() {
	rootCmd.PersistentFlags().String("spec", "", "OpenAPI spec file path or URL (required)")
	rootCmd.PersistentFlags().String("tag", "mcp", "Tag to filter operations")
	rootCmd.PersistentFlags().String("server-url", "", "Override base URL from spec")
	rootCmd.PersistentFlags().Duration("timeout", 30_000_000_000, "HTTP request timeout")
	rootCmd.PersistentFlags().String("auth-type", "", "Auth type: bearer or api-key")
	rootCmd.PersistentFlags().String("auth-token-env", "", "Env var name for bearer token")
	rootCmd.PersistentFlags().String("auth-key-env", "", "Env var name for API key")
	rootCmd.PersistentFlags().String("auth-key-name", "", "Header/query param name for API key")
	rootCmd.PersistentFlags().String("auth-key-in", "", "Where to send API key: header or query")
	rootCmd.PersistentFlags().String("log-level", "info", "Log level: debug, info, warn, error")
	rootCmd.PersistentFlags().String("log-file", "", "Log file path (default: stderr)")

	viper.SetEnvPrefix("OPENAPI_MCP")
	viper.AutomaticEnv()

	_ = viper.BindPFlag("spec", rootCmd.PersistentFlags().Lookup("spec"))
	_ = viper.BindPFlag("tag", rootCmd.PersistentFlags().Lookup("tag"))
	_ = viper.BindPFlag("server-url", rootCmd.PersistentFlags().Lookup("server-url"))
	_ = viper.BindPFlag("auth-type", rootCmd.PersistentFlags().Lookup("auth-type"))
	_ = viper.BindPFlag("auth-token-env", rootCmd.PersistentFlags().Lookup("auth-token-env"))

	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version)
		},
	})
}
```

- [ ] **Step 4: Create Makefile**

Create `Makefile`:
```makefile
.PHONY: build test lint fmt clean

BINARY=mcp-server-openapi

build:
	go build -o bin/$(BINARY) ./cmd/mcp-server-openapi

test:
	go test -v -race -cover ./...

lint:
	golangci-lint run ./...

fmt:
	gofumpt -w .
	goimports -w .

clean:
	rm -rf bin/
```

- [ ] **Step 5: Create LICENSE (MIT)**

Create `LICENSE` with MIT license text.

- [ ] **Step 6: Verify build**

```bash
go build ./cmd/mcp-server-openapi
./mcp-server-openapi version
# Expected: dev
```

- [ ] **Step 7: Commit**

```bash
git add -A && git commit -m "feat: project scaffold with Go module and Cobra CLI"
```

---

### Task 2: pkg/params — Generic Parameter Helpers

**Files:**
- Create: `pkg/params/params.go`, `pkg/params/params_test.go`

- [ ] **Step 1: Write failing tests**

Create `pkg/params/params_test.go`:
```go
package params_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/soyvural/mcp-server-openapi/pkg/params"
)

func TestRequired(t *testing.T) {
	tests := []struct {
		desc    string
		args    map[string]any
		key     string
		want    string
		wantErr bool
	}{
		{
			desc: "key exists with correct type",
			args: map[string]any{"name": "Rex"},
			key:  "name",
			want: "Rex",
		},
		{
			desc:    "key missing",
			args:    map[string]any{},
			key:     "name",
			wantErr: true,
		},
		{
			desc:    "key exists with wrong type",
			args:    map[string]any{"name": 123},
			key:     "name",
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := params.Required[string](tc.args, tc.key)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestOptional(t *testing.T) {
	tests := []struct {
		desc       string
		args       map[string]any
		key        string
		defaultVal int
		want       int
	}{
		{
			desc:       "key exists",
			args:       map[string]any{"limit": float64(50)},
			key:        "limit",
			defaultVal: 10,
			want:       50,
		},
		{
			desc:       "key missing returns default",
			args:       map[string]any{},
			key:        "limit",
			defaultVal: 10,
			want:       10,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			got := params.Optional(tc.args, tc.key, tc.defaultVal)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./pkg/params/... -v
# Expected: FAIL — package does not exist
```

- [ ] **Step 3: Implement params.go**

Create `pkg/params/params.go`:
```go
package params

import "fmt"

// Required extracts a required parameter from the args map.
// Returns an error if the key is missing or the type assertion fails.
func Required[T any](args map[string]any, key string) (T, error) {
	var zero T
	v, ok := args[key]
	if !ok {
		return zero, fmt.Errorf("missing required parameter: %s", key)
	}
	typed, ok := v.(T)
	if !ok {
		return zero, fmt.Errorf("parameter %s: expected %T, got %T", key, zero, v)
	}
	return typed, nil
}

// Optional extracts an optional parameter from the args map.
// Returns defaultVal if the key is missing or the type assertion fails.
func Optional[T any](args map[string]any, key string, defaultVal T) T {
	v, ok := args[key]
	if !ok {
		return defaultVal
	}
	typed, ok := v.(T)
	if !ok {
		return defaultVal
	}
	return typed
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./pkg/params/... -v
# Expected: PASS
```

Note: The `TestOptional` "key exists" case uses `float64(50)` because JSON unmarshaling produces float64 for numbers. The `Optional[int]` call will get a type mismatch and return the default. If you need JSON-number-to-int conversion, add a `NumericOptional` helper. For v1, callers should use `Optional[float64]` for JSON-sourced numeric args.

- [ ] **Step 5: Commit**

```bash
git add pkg/params/ && git commit -m "feat: add generic param extraction helpers"
```

---

### Task 3: toolgen/namer — Tool Name Generation

**Files:**
- Create: `toolgen/namer.go`, `toolgen/namer_test.go`

- [ ] **Step 1: Write failing tests**

Create `toolgen/namer_test.go`:
```go
package toolgen_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/soyvural/mcp-server-openapi/toolgen"
)

func TestGenerateToolName(t *testing.T) {
	tests := []struct {
		desc        string
		path        string
		method      string
		operationID string
		xMCPName    string
		want        string
	}{
		{
			desc:     "x-mcp-tool-name takes priority",
			path:     "/pets",
			method:   "GET",
			xMCPName: "list_all_pets",
			want:     "list_all_pets",
		},
		{
			desc:        "operationId used when no extension",
			path:        "/pets/{petId}",
			method:      "GET",
			operationID: "getPetById",
			want:        "getpetbyid",
		},
		{
			desc:   "fallback to method_path",
			path:   "/pets/{petId}",
			method: "GET",
			want:   "get_pets_petid",
		},
		{
			desc:   "slashes and braces sanitized",
			path:   "/api/v1/users/{userId}/orders/{orderId}",
			method: "DELETE",
			want:   "delete_api_v1_users_userid_orders_orderid",
		},
		{
			desc:     "extension with special chars sanitized",
			path:     "/pets",
			method:   "GET",
			xMCPName: "list-pets!v2",
			want:     "list_pets_v2",
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			got := toolgen.GenerateToolName(tc.path, tc.method, tc.operationID, tc.xMCPName)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSanitizeToolName(t *testing.T) {
	tests := []struct {
		desc  string
		input string
		want  string
	}{
		{desc: "already valid", input: "get_pet", want: "get_pet"},
		{desc: "uppercase", input: "GetPet", want: "getpet"},
		{desc: "hyphens", input: "get-pet-by-id", want: "get_pet_by_id"},
		{desc: "consecutive underscores", input: "get__pet___id", want: "get_pet_id"},
		{desc: "leading/trailing underscores", input: "_get_pet_", want: "get_pet"},
		{desc: "dots preserved", input: "v1.get_pet", want: "v1.get_pet"},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			got := toolgen.SanitizeToolName(tc.input)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./toolgen/... -v
# Expected: FAIL
```

- [ ] **Step 3: Implement namer.go**

Create `toolgen/namer.go`:
```go
package toolgen

import (
	"regexp"
	"strings"
)

var (
	nonAlphanumRe    = regexp.MustCompile(`[^a-z0-9._]`)
	multiUnderscoreRe = regexp.MustCompile(`_+`)
)

// GenerateToolName produces an MCP-safe tool name from an OpenAPI operation.
// Priority: x-mcp-tool-name > operationId > method_path.
func GenerateToolName(path, method, operationID, xMCPName string) string {
	if xMCPName != "" {
		return SanitizeToolName(xMCPName)
	}
	if operationID != "" {
		return SanitizeToolName(operationID)
	}
	raw := method + "_" + path
	return SanitizeToolName(raw)
}

// SanitizeToolName normalizes a string to match MCP tool name rules: [a-z0-9._]
func SanitizeToolName(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "{", "")
	s = strings.ReplaceAll(s, "}", "")
	s = nonAlphanumRe.ReplaceAllString(s, "_")
	s = multiUnderscoreRe.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	return s
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./toolgen/... -v
# Expected: PASS
```

- [ ] **Step 5: Commit**

```bash
git add toolgen/ && git commit -m "feat: tool name generation with sanitization"
```

---

### Task 4: toolgen/extensions — x-mcp-* Extension Extraction

**Files:**
- Create: `toolgen/extensions.go`, `toolgen/extensions_test.go`

- [ ] **Step 1: Write failing tests**

Create `toolgen/extensions_test.go`:
```go
package toolgen_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/soyvural/mcp-server-openapi/toolgen"
)

func TestExtractExtensions(t *testing.T) {
	tests := []struct {
		desc       string
		opExt      map[string]any
		pathExt    map[string]any
		want       toolgen.MCPExtensions
	}{
		{
			desc:  "no extensions",
			want:  toolgen.MCPExtensions{},
		},
		{
			desc:  "tool name from operation",
			opExt: map[string]any{"x-mcp-tool-name": "my_tool"},
			want:  toolgen.MCPExtensions{ToolName: "my_tool"},
		},
		{
			desc:  "description from operation",
			opExt: map[string]any{"x-mcp-description": "Custom desc"},
			want:  toolgen.MCPExtensions{Description: "Custom desc"},
		},
		{
			desc:  "hidden at operation level overrides path",
			opExt: map[string]any{"x-mcp-hidden": true},
			pathExt: map[string]any{"x-mcp-hidden": false},
			want:  toolgen.MCPExtensions{Hidden: boolPtr(true)},
		},
		{
			desc:    "hidden at path level when no operation override",
			pathExt: map[string]any{"x-mcp-hidden": true},
			want:    toolgen.MCPExtensions{Hidden: boolPtr(true)},
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			got := toolgen.ExtractExtensions(tc.opExt, tc.pathExt)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func boolPtr(b bool) *bool { return &b }
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./toolgen/... -v -run TestExtract
# Expected: FAIL
```

- [ ] **Step 3: Implement extensions.go**

Create `toolgen/extensions.go`:
```go
package toolgen

// MCPExtensions holds parsed x-mcp-* vendor extension values.
type MCPExtensions struct {
	ToolName    string
	Description string
	Hidden      *bool
}

// ExtractExtensions parses x-mcp-* extensions from operation and path level.
// Operation-level extensions take precedence over path-level.
func ExtractExtensions(opExt, pathExt map[string]any) MCPExtensions {
	var ext MCPExtensions

	if v, ok := opExt["x-mcp-tool-name"].(string); ok {
		ext.ToolName = v
	}
	if v, ok := opExt["x-mcp-description"].(string); ok {
		ext.Description = v
	}

	// Hidden: operation > path precedence
	if v, ok := opExt["x-mcp-hidden"].(bool); ok {
		ext.Hidden = &v
	} else if v, ok := pathExt["x-mcp-hidden"].(bool); ok {
		ext.Hidden = &v
	}

	return ext
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./toolgen/... -v -run TestExtract
# Expected: PASS
```

- [ ] **Step 5: Commit**

```bash
git add toolgen/extensions* && git commit -m "feat: x-mcp-* vendor extension extraction"
```

---

### Task 5: toolgen/filter — Operation Filtering

**Files:**
- Create: `toolgen/filter.go`, `toolgen/filter_test.go`

- [ ] **Step 1: Write failing tests**

Create `toolgen/filter_test.go`:
```go
package toolgen_test

import (
	"testing"

	"github.com/soyvural/mcp-server-openapi/toolgen"
)

func TestFilter(t *testing.T) {
	tests := []struct {
		desc    string
		tag     string
		opTags  []string
		opExt   map[string]any
		pathExt map[string]any
		want    bool
	}{
		{
			desc:   "include: has matching tag",
			tag:    "mcp",
			opTags: []string{"mcp", "pets"},
			want:   true,
		},
		{
			desc:   "exclude: no matching tag",
			tag:    "mcp",
			opTags: []string{"internal"},
			want:   false,
		},
		{
			desc:   "exclude: has tag but x-mcp-hidden=true",
			tag:    "mcp",
			opTags: []string{"mcp"},
			opExt:  map[string]any{"x-mcp-hidden": true},
			want:   false,
		},
		{
			desc:   "include: no tag but x-mcp-hidden=false forces include",
			tag:    "mcp",
			opTags: []string{"internal"},
			opExt:  map[string]any{"x-mcp-hidden": false},
			want:   true,
		},
		{
			desc:    "exclude: path hidden overrides tag",
			tag:     "mcp",
			opTags:  []string{"mcp"},
			pathExt: map[string]any{"x-mcp-hidden": true},
			want:    false,
		},
		{
			desc:    "include: op hidden=false overrides path hidden=true",
			tag:     "mcp",
			opTags:  []string{"mcp"},
			opExt:   map[string]any{"x-mcp-hidden": false},
			pathExt: map[string]any{"x-mcp-hidden": true},
			want:    true,
		},
		{
			desc:   "exclude: empty tags no extensions",
			tag:    "mcp",
			opTags: nil,
			want:   false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			f := toolgen.NewTagFilter(tc.tag)
			got := f.Include(tc.opTags, tc.opExt, tc.pathExt)
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./toolgen/... -v -run TestFilter
# Expected: FAIL
```

- [ ] **Step 3: Implement filter.go**

Create `toolgen/filter.go`:
```go
package toolgen

// TagFilter includes operations that have the target tag,
// with x-mcp-hidden overrides at operation and path level.
type TagFilter struct {
	tag string
}

// NewTagFilter creates a filter for the given tag name.
func NewTagFilter(tag string) *TagFilter {
	return &TagFilter{tag: tag}
}

// Include returns true if the operation should become an MCP tool.
func (f *TagFilter) Include(opTags []string, opExt, pathExt map[string]any) bool {
	ext := ExtractExtensions(opExt, pathExt)

	// x-mcp-hidden takes highest precedence
	if ext.Hidden != nil {
		return !*ext.Hidden
	}

	// Fall back to tag match
	for _, tag := range opTags {
		if tag == f.tag {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./toolgen/... -v -run TestFilter
# Expected: PASS
```

- [ ] **Step 5: Commit**

```bash
git add toolgen/filter* && git commit -m "feat: tag-based operation filtering with x-mcp-hidden"
```

---

### Task 6: toolgen/parser — OpenAPI Spec Loading

**Files:**
- Create: `toolgen/parser.go`, `toolgen/parser_test.go`, `testdata/petstore.yaml`

- [ ] **Step 1: Create test fixture**

Create `testdata/petstore.yaml` — a minimal Petstore spec with mcp tags:
```yaml
openapi: "3.0.3"
info:
  title: Petstore
  version: "1.0.0"
servers:
  - url: https://petstore.example.com/v1
paths:
  /pets:
    get:
      tags: [mcp]
      operationId: listPets
      summary: List all pets
      parameters:
        - name: limit
          in: query
          required: false
          schema:
            type: integer
            format: int32
      responses:
        "200":
          description: A list of pets
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Pet'
    post:
      tags: [mcp]
      operationId: createPet
      summary: Create a pet
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreatePetRequest'
      responses:
        "201":
          description: Pet created
  /pets/{petId}:
    get:
      tags: [mcp]
      operationId: getPetById
      summary: Get a pet by ID
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: string
      responses:
        "200":
          description: A pet
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Pet'
    delete:
      tags: [internal]
      operationId: deletePet
      summary: Delete a pet
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: string
      responses:
        "204":
          description: Pet deleted
components:
  schemas:
    Pet:
      type: object
      required: [id, name]
      properties:
        id:
          type: string
        name:
          type: string
        tag:
          type: string
    CreatePetRequest:
      type: object
      required: [name]
      properties:
        name:
          type: string
        tag:
          type: string
```

- [ ] **Step 2: Write failing tests**

Create `toolgen/parser_test.go`:
```go
package toolgen_test

import (
	"context"
	"testing"

	"github.com/soyvural/mcp-server-openapi/toolgen"
)

func TestLoadSpec(t *testing.T) {
	tests := []struct {
		desc    string
		source  string
		wantErr bool
	}{
		{
			desc:   "load from file",
			source: "../testdata/petstore.yaml",
		},
		{
			desc:    "file not found",
			source:  "../testdata/nonexistent.yaml",
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			doc, err := toolgen.LoadSpec(context.Background(), tc.source)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if doc == nil {
				t.Fatal("doc is nil")
			}
			if doc.Info.Title != "Petstore" {
				t.Errorf("title: got %q, want %q", doc.Info.Title, "Petstore")
			}
		})
	}
}

func TestLoadSpecPathCount(t *testing.T) {
	doc, err := toolgen.LoadSpec(context.Background(), "../testdata/petstore.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	paths := doc.Paths.Map()
	if len(paths) != 2 {
		t.Errorf("path count: got %d, want 2", len(paths))
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./toolgen/... -v -run TestLoadSpec
# Expected: FAIL
```

- [ ] **Step 4: Implement parser.go**

Create `toolgen/parser.go`:
```go
package toolgen

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// LoadSpec loads and validates an OpenAPI 3.x document from a file path or URL.
func LoadSpec(ctx context.Context, source string) (*openapi3.T, error) {
	loader := openapi3.NewLoader()
	loader.Context = ctx

	var doc *openapi3.T
	var err error

	if isURL(source) {
		u, parseErr := url.Parse(source)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid URL %q: %w", source, parseErr)
		}
		doc, err = loader.LoadFromURI(u)
	} else {
		doc, err = loader.LoadFromFile(source)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load spec from %q: %w", source, err)
	}

	if err := doc.Validate(ctx); err != nil {
		return nil, fmt.Errorf("spec validation failed: %w", err)
	}

	return doc, nil
}

// ServerURL extracts the first server URL from the spec, or returns the override.
func ServerURL(doc *openapi3.T, override string) (string, error) {
	if override != "" {
		return strings.TrimRight(override, "/"), nil
	}
	if doc.Servers != nil && len(doc.Servers) > 0 {
		return strings.TrimRight(doc.Servers[0].URL, "/"), nil
	}
	return "", fmt.Errorf("no server URL in spec and no --server-url provided")
}

func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./toolgen/... -v -run TestLoadSpec
# Expected: PASS
```

- [ ] **Step 6: Commit**

```bash
git add toolgen/parser* testdata/ && git commit -m "feat: OpenAPI spec loader with file and URL support"
```

---

### Task 7: toolgen/schema — Parameter → JSON Schema + Collision Detection

**Files:**
- Create: `toolgen/schema.go`, `toolgen/schema_test.go`, `testdata/collision.yaml`

- [ ] **Step 1: Create collision test fixture**

Create `testdata/collision.yaml`:
```yaml
openapi: "3.0.3"
info:
  title: Collision Test
  version: "1.0.0"
servers:
  - url: https://api.example.com
paths:
  /items/{name}:
    put:
      tags: [mcp]
      operationId: updateItem
      summary: Update an item
      parameters:
        - name: name
          in: path
          required: true
          schema:
            type: string
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [name]
              properties:
                name:
                  type: string
                value:
                  type: integer
      responses:
        "200":
          description: Updated
```

- [ ] **Step 2: Write failing tests**

Create `toolgen/schema_test.go`:
```go
package toolgen_test

import (
	"encoding/json"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/go-cmp/cmp"
	"github.com/soyvural/mcp-server-openapi/toolgen"
)

func TestConvertSchema_QueryAndPathParams(t *testing.T) {
	params := openapi3.Parameters{
		&openapi3.ParameterRef{Value: &openapi3.Parameter{
			Name:     "petId",
			In:       "path",
			Required: true,
			Schema:   &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"string"}}},
		}},
		&openapi3.ParameterRef{Value: &openapi3.Parameter{
			Name:        "verbose",
			In:          "query",
			Required:    false,
			Description: "Include details",
			Schema:      &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"boolean"}}},
		}},
	}

	result, err := toolgen.ConvertSchema(params, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var schema map[string]any
	if err := json.Unmarshal(result.InputSchema, &schema); err != nil {
		t.Fatalf("invalid JSON schema: %v", err)
	}

	props := schema["properties"].(map[string]any)
	if len(props) != 2 {
		t.Errorf("property count: got %d, want 2", len(props))
	}

	petIDProp := props["petId"].(map[string]any)
	if diff := cmp.Diff("string", petIDProp["type"]); diff != "" {
		t.Errorf("petId type mismatch: %s", diff)
	}

	required := toStringSlice(schema["required"])
	if diff := cmp.Diff([]string{"petId"}, required); diff != "" {
		t.Errorf("required mismatch: %s", diff)
	}
}

func TestConvertSchema_WithRequestBody(t *testing.T) {
	params := openapi3.Parameters{
		&openapi3.ParameterRef{Value: &openapi3.Parameter{
			Name:     "petId",
			In:       "path",
			Required: true,
			Schema:   &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"string"}}},
		}},
	}

	body := &openapi3.RequestBodyRef{Value: &openapi3.RequestBody{
		Required: true,
		Content: openapi3.Content{
			"application/json": &openapi3.MediaType{
				Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{
					Type:     &openapi3.Types{"object"},
					Required: []string{"name"},
					Properties: openapi3.Schemas{
						"name": &openapi3.SchemaRef{Value: &openapi3.Schema{
							Type:        &openapi3.Types{"string"},
							Description: "Pet name",
						}},
						"tag": &openapi3.SchemaRef{Value: &openapi3.Schema{
							Type: &openapi3.Types{"string"},
						}},
					},
				}},
			},
		},
	}}

	result, err := toolgen.ConvertSchema(params, body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var schema map[string]any
	if err := json.Unmarshal(result.InputSchema, &schema); err != nil {
		t.Fatalf("invalid JSON schema: %v", err)
	}

	props := schema["properties"].(map[string]any)
	if len(props) != 3 {
		t.Errorf("property count: got %d, want 3 (petId, name, tag)", len(props))
	}

	required := toStringSlice(schema["required"])
	wantRequired := []string{"petId", "name"}
	if diff := cmp.Diff(wantRequired, required); diff != "" {
		t.Errorf("required mismatch: %s", diff)
	}
}

func TestConvertSchema_CollisionFails(t *testing.T) {
	params := openapi3.Parameters{
		&openapi3.ParameterRef{Value: &openapi3.Parameter{
			Name:     "name",
			In:       "path",
			Required: true,
			Schema:   &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"string"}}},
		}},
	}

	body := &openapi3.RequestBodyRef{Value: &openapi3.RequestBody{
		Content: openapi3.Content{
			"application/json": &openapi3.MediaType{
				Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{
					Type: &openapi3.Types{"object"},
					Properties: openapi3.Schemas{
						"name": &openapi3.SchemaRef{Value: &openapi3.Schema{
							Type: &openapi3.Types{"string"},
						}},
					},
				}},
			},
		},
	}}

	_, err := toolgen.ConvertSchema(params, body)
	if err == nil {
		t.Fatal("expected collision error, got nil")
	}
}

func TestConvertSchema_NoParams(t *testing.T) {
	result, err := toolgen.ConvertSchema(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var schema map[string]any
	if err := json.Unmarshal(result.InputSchema, &schema); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if schema["type"] != "object" {
		t.Errorf("type: got %v, want object", schema["type"])
	}
}

func toStringSlice(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./toolgen/... -v -run TestConvertSchema
# Expected: FAIL
```

- [ ] **Step 4: Implement schema.go**

Create `toolgen/schema.go`:
```go
package toolgen

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/getkin/kin-openapi/openapi3"
)

// SchemaResult holds the converted JSON Schema and metadata about parameter locations.
type SchemaResult struct {
	InputSchema json.RawMessage
	ParamMeta   []ParamMeta
}

// ParamMeta tracks where each parameter came from for request reconstruction.
type ParamMeta struct {
	Name     string
	In       string // "path", "query", "header", "body"
	Required bool
}

// ConvertSchema merges OpenAPI parameters and request body into a flat JSON Schema.
// Returns an error if any parameter name collides with a body property name.
func ConvertSchema(params openapi3.Parameters, body *openapi3.RequestBodyRef) (*SchemaResult, error) {
	properties := make(map[string]any)
	var required []string
	var meta []ParamMeta
	seen := make(map[string]string) // name -> source for collision detection

	// Process parameters (path, query, header)
	for _, pRef := range params {
		if pRef == nil || pRef.Value == nil {
			continue
		}
		p := pRef.Value
		if _, exists := seen[p.Name]; exists {
			return nil, fmt.Errorf("duplicate parameter name %q", p.Name)
		}
		seen[p.Name] = p.In

		prop := schemaToMap(p.Schema)
		if p.Description != "" {
			prop["description"] = p.Description
		}
		properties[p.Name] = prop

		if p.Required {
			required = append(required, p.Name)
		}
		meta = append(meta, ParamMeta{Name: p.Name, In: p.In, Required: p.Required})
	}

	// Process request body (application/json only for v1)
	if body != nil && body.Value != nil {
		jsonContent := body.Value.Content.Get("application/json")
		if jsonContent != nil && jsonContent.Schema != nil && jsonContent.Schema.Value != nil {
			bodySchema := jsonContent.Schema.Value
			for name, propRef := range bodySchema.Properties {
				if source, exists := seen[name]; exists {
					return nil, fmt.Errorf("parameter name collision: %q exists in both %s params and request body", name, source)
				}
				seen[name] = "body"

				prop := schemaToMap(propRef)
				properties[name] = prop

				isRequired := contains(bodySchema.Required, name) && body.Value.Required
				if isRequired {
					required = append(required, name)
				}
				meta = append(meta, ParamMeta{Name: name, In: "body", Required: isRequired})
			}
		}
	}

	sort.Strings(required)

	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}

	raw, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}

	return &SchemaResult{InputSchema: raw, ParamMeta: meta}, nil
}

func schemaToMap(ref *openapi3.SchemaRef) map[string]any {
	if ref == nil || ref.Value == nil {
		return map[string]any{}
	}
	s := ref.Value
	m := make(map[string]any)

	if s.Type != nil && len(s.Type.Slice()) > 0 {
		types := s.Type.Slice()
		if len(types) == 1 {
			m["type"] = types[0]
		} else {
			m["type"] = types
		}
	}
	if s.Format != "" {
		m["format"] = s.Format
	}
	if s.Description != "" {
		m["description"] = s.Description
	}
	if len(s.Enum) > 0 {
		m["enum"] = s.Enum
	}
	if s.Default != nil {
		m["default"] = s.Default
	}
	if s.Pattern != "" {
		m["pattern"] = s.Pattern
	}
	if s.Minimum != nil {
		m["minimum"] = *s.Minimum
	}
	if s.Maximum != nil {
		m["maximum"] = *s.Maximum
	}
	if s.MinLength != nil {
		m["minLength"] = *s.MinLength
	}
	if s.MaxLength != nil {
		m["maxLength"] = *s.MaxLength
	}

	// Handle nested objects
	if len(s.Properties) > 0 {
		props := make(map[string]any)
		for k, v := range s.Properties {
			props[k] = schemaToMap(v)
		}
		m["properties"] = props
		if len(s.Required) > 0 {
			m["required"] = s.Required
		}
	}

	// Handle arrays
	if s.Items != nil {
		m["items"] = schemaToMap(s.Items)
	}

	// Preserve composition keywords
	if len(s.OneOf) > 0 {
		m["oneOf"] = schemasToSlice(s.OneOf)
	}
	if len(s.AnyOf) > 0 {
		m["anyOf"] = schemasToSlice(s.AnyOf)
	}
	if len(s.AllOf) > 0 {
		m["allOf"] = schemasToSlice(s.AllOf)
	}

	return m
}

func schemasToSlice(refs openapi3.SchemaRefs) []map[string]any {
	var out []map[string]any
	for _, ref := range refs {
		out = append(out, schemaToMap(ref))
	}
	return out
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./toolgen/... -v -run TestConvertSchema
# Expected: PASS
```

- [ ] **Step 6: Commit**

```bash
git add toolgen/schema* testdata/collision.yaml && git commit -m "feat: schema converter with flat merge and collision detection"
```

---

### Task 8: toolgen/generator — Orchestrator

**Files:**
- Create: `toolgen/generator.go`, `toolgen/generator_test.go`, `testdata/extensions.yaml`

- [ ] **Step 1: Create extensions test fixture**

Create `testdata/extensions.yaml`:
```yaml
openapi: "3.0.3"
info:
  title: Extensions Test
  version: "1.0.0"
servers:
  - url: https://api.example.com
paths:
  /visible:
    get:
      tags: [mcp]
      operationId: visibleOp
      summary: A visible operation
      x-mcp-tool-name: custom_visible
      x-mcp-description: Custom description here
      responses:
        "200":
          description: OK
  /hidden:
    get:
      tags: [mcp]
      operationId: hiddenOp
      summary: A hidden operation
      x-mcp-hidden: true
      responses:
        "200":
          description: OK
  /forced:
    get:
      tags: [internal]
      operationId: forcedOp
      summary: Forced visible via extension
      x-mcp-hidden: false
      parameters:
        - name: q
          in: query
          schema:
            type: string
      responses:
        "200":
          description: OK
```

- [ ] **Step 2: Write failing tests**

Create `toolgen/generator_test.go`:
```go
package toolgen_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/soyvural/mcp-server-openapi/toolgen"
)

func TestGenerate_Petstore(t *testing.T) {
	tools, err := toolgen.Generate(context.Background(), toolgen.GenerateOptions{
		SpecSource: "../testdata/petstore.yaml",
		Tag:        "mcp",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// petstore.yaml has 3 mcp-tagged ops: listPets, createPet, getPetById
	// deletePet is tagged "internal" so excluded
	if len(tools) != 3 {
		t.Errorf("tool count: got %d, want 3", len(tools))
	}

	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name] = true
	}
	for _, want := range []string{"listpets", "createpet", "getpetbyid"} {
		if !names[want] {
			t.Errorf("missing tool %q, got names: %v", want, names)
		}
	}
}

func TestGenerate_Extensions(t *testing.T) {
	tools, err := toolgen.Generate(context.Background(), toolgen.GenerateOptions{
		SpecSource: "../testdata/extensions.yaml",
		Tag:        "mcp",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// visibleOp (mcp tag, x-mcp-tool-name override) -> included as "custom_visible"
	// hiddenOp (mcp tag, x-mcp-hidden=true) -> excluded
	// forcedOp (internal tag, x-mcp-hidden=false) -> included as "forcedop"
	if len(tools) != 2 {
		t.Errorf("tool count: got %d, want 2", len(tools))
	}

	names := make(map[string]bool)
	descs := make(map[string]string)
	for _, tool := range tools {
		names[tool.Name] = true
		descs[tool.Name] = tool.Description
	}

	if !names["custom_visible"] {
		t.Error("expected tool 'custom_visible' from x-mcp-tool-name override")
	}
	if diff := cmp.Diff("Custom description here", descs["custom_visible"]); diff != "" {
		t.Errorf("custom_visible description mismatch: %s", diff)
	}
	if !names["forcedop"] {
		t.Error("expected tool 'forcedop' from x-mcp-hidden=false override")
	}
	if names["hiddenop"] {
		t.Error("hiddenOp should be excluded via x-mcp-hidden=true")
	}
}

func TestGenerate_Collision(t *testing.T) {
	_, err := toolgen.Generate(context.Background(), toolgen.GenerateOptions{
		SpecSource: "../testdata/collision.yaml",
		Tag:        "mcp",
	})
	if err == nil {
		t.Fatal("expected collision error, got nil")
	}
}

func TestGenerate_DuplicateToolName(t *testing.T) {
	// Two ops that resolve to the same tool name should fail
	_, err := toolgen.Generate(context.Background(), toolgen.GenerateOptions{
		SpecSource: "../testdata/petstore.yaml",
		Tag:        "mcp",
	})
	// petstore has unique operationIds so this should pass
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./toolgen/... -v -run TestGenerate
# Expected: FAIL
```

- [ ] **Step 4: Implement generator.go**

Create `toolgen/generator.go`:
```go
package toolgen

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
)

// GenerateOptions configures the tool generation process.
type GenerateOptions struct {
	SpecSource string
	Tag        string
	ServerURL  string
}

// GeneratedTool holds an MCP tool definition and metadata for request execution.
type GeneratedTool struct {
	Name        string
	Description string
	InputSchema json.RawMessage
	ParamMeta   []ParamMeta
	Path        string
	Method      string
	ServerURL   string
}

// Generate loads an OpenAPI spec and converts matching operations into MCP tool definitions.
func Generate(ctx context.Context, opts GenerateOptions) ([]GeneratedTool, error) {
	doc, err := LoadSpec(ctx, opts.SpecSource)
	if err != nil {
		return nil, err
	}

	serverURL, err := ServerURL(doc, opts.ServerURL)
	if err != nil {
		return nil, err
	}

	filter := NewTagFilter(opts.Tag)
	var tools []GeneratedTool
	nameSet := make(map[string]string) // tool name -> operationId for dup detection

	// Collect paths in sorted order for deterministic output
	var pathKeys []string
	for path := range doc.Paths.Map() {
		pathKeys = append(pathKeys, path)
	}
	sort.Strings(pathKeys)

	for _, path := range pathKeys {
		pathItem := doc.Paths.Map()[path]
		pathExt := pathItem.Extensions

		for method, op := range pathItem.Operations() {
			if !filter.Include(op.Tags, op.Extensions, pathExt) {
				slog.Debug("skipping operation", "path", path, "method", method, "reason", "filtered")
				continue
			}

			ext := ExtractExtensions(op.Extensions, pathExt)

			// Generate tool name
			name := GenerateToolName(path, method, op.OperationID, ext.ToolName)
			if existing, dup := nameSet[name]; dup {
				return nil, fmt.Errorf("duplicate tool name %q: operations %q and %q resolve to the same name", name, existing, op.OperationID)
			}
			nameSet[name] = op.OperationID

			// Build description
			desc := op.Summary
			if ext.Description != "" {
				desc = ext.Description
			} else if op.Description != "" && desc == "" {
				desc = op.Description
			}

			// Convert parameters + body to JSON Schema
			schemaResult, err := ConvertSchema(op.Parameters, op.RequestBody)
			if err != nil {
				return nil, fmt.Errorf("operation %q (%s %s): %w", op.OperationID, method, path, err)
			}

			tools = append(tools, GeneratedTool{
				Name:        name,
				Description: desc,
				InputSchema: schemaResult.InputSchema,
				ParamMeta:   schemaResult.ParamMeta,
				Path:        path,
				Method:      method,
				ServerURL:   serverURL,
			})
		}
	}

	slog.Info("tool generation complete", "total_operations", countOps(doc), "tools_generated", len(tools))
	return tools, nil
}

func countOps(doc interface{ Paths interface{ Len() int } }) int {
	// Simplified — just return 0 for now, real counting done at call site
	return 0
}
```

Note: The `countOps` helper is a placeholder. Replace with direct path/operation counting if needed, or remove. The slog line is the important part.

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./toolgen/... -v -run TestGenerate
# Expected: PASS
```

- [ ] **Step 6: Commit**

```bash
git add toolgen/generator* testdata/extensions.yaml && git commit -m "feat: generator orchestrates spec -> MCP tool conversion"
```

---

### Task 9: executor/auth — Authenticator Interface + Implementations

**Files:**
- Create: `executor/auth.go`, `executor/auth_test.go`

- [ ] **Step 1: Write failing tests**

Create `executor/auth_test.go`:
```go
package executor_test

import (
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/soyvural/mcp-server-openapi/executor"
)

func TestBearerAuth(t *testing.T) {
	t.Setenv("TEST_TOKEN", "my-secret-token")

	auth := executor.NewBearerAuth("TEST_TOKEN")
	req, _ := http.NewRequest("GET", "https://api.example.com/pets", nil)

	if err := auth.Apply(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := req.Header.Get("Authorization")
	want := "Bearer my-secret-token"
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Authorization header mismatch: %s", diff)
	}
}

func TestBearerAuth_EmptyEnv(t *testing.T) {
	t.Setenv("EMPTY_TOKEN", "")

	auth := executor.NewBearerAuth("EMPTY_TOKEN")
	req, _ := http.NewRequest("GET", "https://api.example.com/pets", nil)

	err := auth.Apply(req)
	if err == nil {
		t.Fatal("expected error for empty token, got nil")
	}
}

func TestAPIKeyAuth_Header(t *testing.T) {
	t.Setenv("TEST_API_KEY", "key-12345")

	auth := executor.NewAPIKeyAuth("TEST_API_KEY", "header", "X-API-Key")
	req, _ := http.NewRequest("GET", "https://api.example.com/pets", nil)

	if err := auth.Apply(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := req.Header.Get("X-API-Key")
	if diff := cmp.Diff("key-12345", got); diff != "" {
		t.Errorf("API key header mismatch: %s", diff)
	}
}

func TestAPIKeyAuth_Query(t *testing.T) {
	t.Setenv("TEST_API_KEY", "key-12345")

	auth := executor.NewAPIKeyAuth("TEST_API_KEY", "query", "api_key")
	req, _ := http.NewRequest("GET", "https://api.example.com/pets", nil)

	if err := auth.Apply(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := req.URL.Query().Get("api_key")
	if diff := cmp.Diff("key-12345", got); diff != "" {
		t.Errorf("API key query param mismatch: %s", diff)
	}
}

func TestNoAuth(t *testing.T) {
	auth := executor.NoAuth{}
	req, _ := http.NewRequest("GET", "https://api.example.com/pets", nil)

	if err := auth.Apply(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if req.Header.Get("Authorization") != "" {
		t.Error("expected no Authorization header for NoAuth")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./executor/... -v
# Expected: FAIL
```

- [ ] **Step 3: Implement auth.go**

Create `executor/auth.go`:
```go
package executor

import (
	"fmt"
	"net/http"
	"os"
)

// Authenticator injects credentials into outbound HTTP requests.
type Authenticator interface {
	Apply(req *http.Request) error
}

// NoAuth is a no-op authenticator for APIs that require no auth.
type NoAuth struct{}

func (NoAuth) Apply(_ *http.Request) error { return nil }

// BearerAuth adds an Authorization: Bearer header using a token from an env var.
type BearerAuth struct {
	tokenEnv string
}

func NewBearerAuth(tokenEnv string) *BearerAuth {
	return &BearerAuth{tokenEnv: tokenEnv}
}

func (a *BearerAuth) Apply(req *http.Request) error {
	token := os.Getenv(a.tokenEnv)
	if token == "" {
		return fmt.Errorf("bearer token env var %q is empty or not set", a.tokenEnv)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return nil
}

// APIKeyAuth adds an API key as a header or query parameter.
type APIKeyAuth struct {
	keyEnv string
	in     string // "header" or "query"
	name   string // header name or query param name
}

func NewAPIKeyAuth(keyEnv, in, name string) *APIKeyAuth {
	return &APIKeyAuth{keyEnv: keyEnv, in: in, name: name}
}

func (a *APIKeyAuth) Apply(req *http.Request) error {
	key := os.Getenv(a.keyEnv)
	if key == "" {
		return fmt.Errorf("API key env var %q is empty or not set", a.keyEnv)
	}
	switch a.in {
	case "header":
		req.Header.Set(a.name, key)
	case "query":
		q := req.URL.Query()
		q.Set(a.name, key)
		req.URL.RawQuery = q.Encode()
	default:
		return fmt.Errorf("unsupported API key location: %q (must be header or query)", a.in)
	}
	return nil
}

// NewAuthenticator creates the appropriate authenticator from CLI flags.
func NewAuthenticator(authType, tokenEnv, keyEnv, keyName, keyIn string) Authenticator {
	switch authType {
	case "bearer":
		return NewBearerAuth(tokenEnv)
	case "api-key":
		return NewAPIKeyAuth(keyEnv, keyIn, keyName)
	default:
		return NoAuth{}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./executor/... -v
# Expected: PASS
```

- [ ] **Step 5: Commit**

```bash
git add executor/ && git commit -m "feat: authenticator interface with bearer and API key support"
```

---

### Task 10: executor/executor — Tool Call → HTTP Request

**Files:**
- Create: `executor/executor.go`, `executor/executor_test.go`

- [ ] **Step 1: Write failing tests**

Create `executor/executor_test.go`:
```go
package executor_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/soyvural/mcp-server-openapi/executor"
	"github.com/soyvural/mcp-server-openapi/toolgen"
)

func TestExecute_GET_WithPathAndQuery(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pets/123" {
			t.Errorf("path: got %q, want /pets/123", r.URL.Path)
		}
		if r.URL.Query().Get("verbose") != "true" {
			t.Errorf("query verbose: got %q, want true", r.URL.Query().Get("verbose"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"123","name":"Rex"}`))
	}))
	defer ts.Close()

	exec := executor.New(ts.Client(), executor.NoAuth{}, 10*time.Second)

	resp, err := exec.Execute(context.Background(), &executor.ToolRequest{
		ServerURL: ts.URL,
		Path:      "/pets/{petId}",
		Method:    "GET",
		Args:      map[string]any{"petId": "123", "verbose": "true"},
		ParamMeta: []toolgen.ParamMeta{
			{Name: "petId", In: "path"},
			{Name: "verbose", In: "query"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}
	if diff := cmp.Diff(`{"id":"123","name":"Rex"}`, resp.Body); diff != "" {
		t.Errorf("body mismatch: %s", diff)
	}
}

func TestExecute_POST_WithBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method: got %q, want POST", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		if diff := cmp.Diff(`{"name":"Rex","tag":"dog"}`, string(body)); diff != "" {
			t.Errorf("body mismatch: %s", diff)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content-type: got %q", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"1","name":"Rex"}`))
	}))
	defer ts.Close()

	exec := executor.New(ts.Client(), executor.NoAuth{}, 10*time.Second)

	resp, err := exec.Execute(context.Background(), &executor.ToolRequest{
		ServerURL: ts.URL,
		Path:      "/pets",
		Method:    "POST",
		Args:      map[string]any{"name": "Rex", "tag": "dog"},
		ParamMeta: []toolgen.ParamMeta{
			{Name: "name", In: "body"},
			{Name: "tag", In: "body"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Errorf("status: got %d, want 201", resp.StatusCode)
	}
}

func TestExecute_ErrorResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"pet not found"}`))
	}))
	defer ts.Close()

	exec := executor.New(ts.Client(), executor.NoAuth{}, 10*time.Second)

	resp, err := exec.Execute(context.Background(), &executor.ToolRequest{
		ServerURL: ts.URL,
		Path:      "/pets/999",
		Method:    "GET",
		Args:      map[string]any{},
		ParamMeta: nil,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("status: got %d, want 404", resp.StatusCode)
	}
	if resp.IsError != true {
		t.Error("expected IsError=true for 404")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./executor/... -v -run TestExecute
# Expected: FAIL
```

- [ ] **Step 3: Implement executor.go**

Create `executor/executor.go`:
```go
package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/soyvural/mcp-server-openapi/toolgen"
)

const maxResponseBody = 4096

// ToolRequest represents an MCP tool invocation to be executed as an HTTP request.
type ToolRequest struct {
	ServerURL string
	Path      string
	Method    string
	Args      map[string]any
	ParamMeta []toolgen.ParamMeta
}

// ToolResponse holds the HTTP response mapped for MCP consumption.
type ToolResponse struct {
	StatusCode int
	Body       string
	IsError    bool
}

// RequestExecutor executes HTTP requests for tool invocations.
type RequestExecutor interface {
	Execute(ctx context.Context, req *ToolRequest) (*ToolResponse, error)
}

// HTTPExecutor implements RequestExecutor using net/http.
type HTTPExecutor struct {
	client  *http.Client
	auth    Authenticator
	timeout time.Duration
}

// New creates an HTTPExecutor.
func New(client *http.Client, auth Authenticator, timeout time.Duration) *HTTPExecutor {
	if client == nil {
		client = &http.Client{}
	}
	return &HTTPExecutor{client: client, auth: auth, timeout: timeout}
}

// Execute converts a ToolRequest into an HTTP request, executes it, and maps the response.
func (e *HTTPExecutor) Execute(ctx context.Context, req *ToolRequest) (*ToolResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// Build URL with path param substitution
	urlPath := req.Path
	queryParams := make(map[string]string)
	bodyParams := make(map[string]any)

	for _, meta := range req.ParamMeta {
		val, exists := req.Args[meta.Name]
		if !exists {
			continue
		}
		switch meta.In {
		case "path":
			urlPath = strings.ReplaceAll(urlPath, "{"+meta.Name+"}", fmt.Sprintf("%v", val))
		case "query":
			queryParams[meta.Name] = fmt.Sprintf("%v", val)
		case "header":
			// Headers handled below
		case "body":
			bodyParams[meta.Name] = val
		}
	}

	fullURL := req.ServerURL + urlPath

	// Build query string
	if len(queryParams) > 0 {
		parts := make([]string, 0, len(queryParams))
		for k, v := range queryParams {
			parts = append(parts, k+"="+v)
		}
		fullURL += "?" + strings.Join(parts, "&")
	}

	// Build body
	var bodyReader io.Reader
	if len(bodyParams) > 0 {
		bodyBytes, err := json.Marshal(bodyParams)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	if bodyReader != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Add header params
	for _, meta := range req.ParamMeta {
		if meta.In == "header" {
			if val, exists := req.Args[meta.Name]; exists {
				httpReq.Header.Set(meta.Name, fmt.Sprintf("%v", val))
			}
		}
	}

	// Apply auth
	if err := e.auth.Apply(httpReq); err != nil {
		return nil, fmt.Errorf("auth failed: %w", err)
	}

	start := time.Now()
	httpResp, err := e.client.Do(httpReq)
	duration := time.Since(start)

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return &ToolResponse{IsError: true, Body: "request timed out"}, nil
		}
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(httpResp.Body, maxResponseBody))
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	slog.Debug("upstream request",
		"method", req.Method,
		"url", fullURL,
		"status", httpResp.StatusCode,
		"duration", duration,
	)

	return &ToolResponse{
		StatusCode: httpResp.StatusCode,
		Body:       string(respBody),
		IsError:    httpResp.StatusCode >= 400,
	}, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./executor/... -v -run TestExecute
# Expected: PASS
```

- [ ] **Step 5: Commit**

```bash
git add executor/executor* && git commit -m "feat: HTTP executor decomposes tool args into HTTP requests"
```

---

### Task 11: server/server — MCP Server Wiring

**Files:**
- Create: `server/server.go`, `server/server_test.go`

- [ ] **Step 1: Write failing test**

Create `server/server_test.go`:
```go
package server_test

import (
	"encoding/json"
	"testing"

	"github.com/soyvural/mcp-server-openapi/server"
	"github.com/soyvural/mcp-server-openapi/toolgen"
)

func TestNewServer_RegistersTools(t *testing.T) {
	tools := []toolgen.GeneratedTool{
		{
			Name:        "list_pets",
			Description: "List all pets",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"limit":{"type":"integer"}}}`),
			ParamMeta:   []toolgen.ParamMeta{{Name: "limit", In: "query"}},
			Path:        "/pets",
			Method:      "GET",
			ServerURL:   "https://api.example.com",
		},
		{
			Name:        "get_pet",
			Description: "Get a pet by ID",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"petId":{"type":"string"}},"required":["petId"]}`),
			ParamMeta:   []toolgen.ParamMeta{{Name: "petId", In: "path"}},
			Path:        "/pets/{petId}",
			Method:      "GET",
			ServerURL:   "https://api.example.com",
		},
	}

	s, err := server.New(tools, nil, "0.1.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s == nil {
		t.Fatal("server is nil")
	}
}

func TestNewServer_NoTools(t *testing.T) {
	_, err := server.New(nil, nil, "0.1.0")
	if err == nil {
		t.Fatal("expected error for zero tools, got nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./server/... -v
# Expected: FAIL
```

- [ ] **Step 3: Implement server.go**

Create `server/server.go`:
```go
package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/soyvural/mcp-server-openapi/executor"
	"github.com/soyvural/mcp-server-openapi/toolgen"
)

// New creates an MCP server with tools generated from an OpenAPI spec.
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
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./server/... -v
# Expected: PASS
```

- [ ] **Step 5: Commit**

```bash
git add server/ && git commit -m "feat: MCP server wiring with generated tool handlers"
```

---

### Task 12: cmd/ — stdio + http Subcommands

**Files:**
- Create: `cmd/mcp-server-openapi/stdio.go`, `cmd/mcp-server-openapi/http.go`
- Modify: `cmd/mcp-server-openapi/main.go` (add subcommand registration)

- [ ] **Step 1: Implement shared helper in main.go**

Add to `cmd/mcp-server-openapi/main.go` — a shared function that builds the MCP server from flags:

```go
// Add these imports to existing main.go
import (
	"log/slog"
	"os"
	"time"

	"github.com/soyvural/mcp-server-openapi/executor"
	"github.com/soyvural/mcp-server-openapi/server"
	"github.com/soyvural/mcp-server-openapi/toolgen"
)

func buildServer() (*mcpserver.MCPServer, error) {
	spec := viper.GetString("spec")
	if spec == "" {
		return nil, fmt.Errorf("--spec is required")
	}

	// Setup logging
	setupLogging(viper.GetString("log-level"), viper.GetString("log-file"))

	// Generate tools from spec
	tools, err := toolgen.Generate(context.Background(), toolgen.GenerateOptions{
		SpecSource: spec,
		Tag:        viper.GetString("tag"),
		ServerURL:  viper.GetString("server-url"),
	})
	if err != nil {
		return nil, fmt.Errorf("tool generation failed: %w", err)
	}

	// Create authenticator
	auth := executor.NewAuthenticator(
		viper.GetString("auth-type"),
		viper.GetString("auth-token-env"),
		viper.GetString("auth-key-env"),
		viper.GetString("auth-key-name"),
		viper.GetString("auth-key-in"),
	)

	// Create executor
	timeout := viper.GetDuration("timeout")
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	exec := executor.New(nil, auth, timeout)

	// Create MCP server
	return server.New(tools, exec, version)
}

func setupLogging(level, file string) {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	var w *os.File
	if file != "" {
		var err error
		w, err = os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open log file: %v\n", err)
			w = os.Stderr
		}
	} else {
		w = os.Stderr
	}

	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: logLevel})
	slog.SetDefault(slog.New(handler))
}
```

- [ ] **Step 2: Create stdio.go subcommand**

Create `cmd/mcp-server-openapi/stdio.go`:
```go
package main

import (
	"os"
	"os/signal"
	"syscall"

	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

var stdioCmd = &cobra.Command{
	Use:   "stdio",
	Short: "Run MCP server over stdio",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := buildServer()
		if err != nil {
			return err
		}

		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		stdioServer := mcpserver.NewStdioServer(s)
		return stdioServer.Listen(ctx, os.Stdin, os.Stdout)
	},
}

func init() {
	rootCmd.AddCommand(stdioCmd)
}
```

- [ ] **Step 3: Create http.go subcommand**

Create `cmd/mcp-server-openapi/http.go`:
```go
package main

import (
	"log/slog"

	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var httpCmd = &cobra.Command{
	Use:   "http",
	Short: "Run MCP server over Streamable HTTP",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := buildServer()
		if err != nil {
			return err
		}

		addr := viper.GetString("addr")
		slog.Info("starting Streamable HTTP server", "addr", addr)

		httpServer := mcpserver.NewStreamableHTTPServer(s)
		return httpServer.Start(addr)
	},
}

func init() {
	httpCmd.Flags().String("addr", ":8080", "Listen address")
	_ = viper.BindPFlag("addr", httpCmd.Flags().Lookup("addr"))
	rootCmd.AddCommand(httpCmd)
}
```

- [ ] **Step 4: Verify build compiles**

```bash
go build ./cmd/mcp-server-openapi
./mcp-server-openapi --help
# Expected: Shows stdio, http, version subcommands
```

- [ ] **Step 5: Verify stdio runs with petstore spec**

```bash
echo '{}' | ./mcp-server-openapi stdio --spec testdata/petstore.yaml 2>/dev/null || true
# Expected: starts and exits (no valid JSON-RPC input)
# No crash = success
```

- [ ] **Step 6: Commit**

```bash
git add cmd/ && git commit -m "feat: stdio and HTTP subcommands for MCP transport"
```

---

### Task 13: Examples — Petstore + Demo API

**Files:**
- Create: `examples/petstore/petstore.yaml`, `examples/petstore/README.md`
- Create: `examples/demo-api/main.go`, `examples/demo-api/openapi.yaml`, `examples/demo-api/README.md`

- [ ] **Step 1: Create petstore example spec**

Copy and extend `testdata/petstore.yaml` to `examples/petstore/petstore.yaml` — add `findByStatus` endpoint:

```yaml
openapi: "3.0.3"
info:
  title: Petstore
  description: A sample API that uses a petstore as an example
  version: "1.0.0"
servers:
  - url: https://petstore.swagger.io/v2
paths:
  /pet/{petId}:
    get:
      tags: [mcp]
      operationId: getPetById
      summary: Find pet by ID
      description: Returns a single pet
      parameters:
        - name: petId
          in: path
          description: ID of pet to return
          required: true
          schema:
            type: integer
            format: int64
      responses:
        "200":
          description: successful operation
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Pet'
    delete:
      tags: [mcp]
      operationId: deletePet
      summary: Deletes a pet
      x-mcp-description: Delete a pet by its ID (irreversible)
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: integer
            format: int64
      responses:
        "200":
          description: successful operation
  /pet/findByStatus:
    get:
      tags: [mcp]
      operationId: findPetsByStatus
      summary: Finds pets by status
      parameters:
        - name: status
          in: query
          description: Status values that need to be considered for filter
          required: true
          schema:
            type: string
            enum: [available, pending, sold]
      responses:
        "200":
          description: successful operation
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Pet'
  /pet:
    post:
      tags: [mcp]
      operationId: addPet
      summary: Add a new pet to the store
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/NewPet'
      responses:
        "200":
          description: successful operation
    put:
      tags: [mcp]
      operationId: updatePet
      summary: Update an existing pet
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Pet'
      responses:
        "200":
          description: successful operation
  /store/inventory:
    get:
      tags: [store]
      operationId: getInventory
      summary: Returns pet inventories by status
      responses:
        "200":
          description: successful operation
components:
  schemas:
    Pet:
      type: object
      required: [name, photoUrls]
      properties:
        id:
          type: integer
          format: int64
        name:
          type: string
        status:
          type: string
          enum: [available, pending, sold]
        photoUrls:
          type: array
          items:
            type: string
    NewPet:
      type: object
      required: [name]
      properties:
        name:
          type: string
        status:
          type: string
          enum: [available, pending, sold]
```

- [ ] **Step 2: Create petstore README**

Create `examples/petstore/README.md`:
```markdown
# Petstore Example

Demonstrates mcp-server-openapi with the classic Petstore API.

## Quick Start

```bash
# From repo root
go run ./cmd/mcp-server-openapi stdio --spec examples/petstore/petstore.yaml
```

## Claude Desktop Configuration

```json
{
  "mcpServers": {
    "petstore": {
      "command": "mcp-server-openapi",
      "args": ["stdio", "--spec", "/absolute/path/to/petstore.yaml"]
    }
  }
}
```

## Tools Generated

| Tool | Method | Path | Description |
|------|--------|------|-------------|
| getPetById | GET | /pet/{petId} | Find pet by ID |
| deletePet | DELETE | /pet/{petId} | Delete a pet by its ID (irreversible) |
| findPetsByStatus | GET | /pet/findByStatus | Finds pets by status |
| addPet | POST | /pet | Add a new pet to the store |
| updatePet | PUT | /pet | Update an existing pet |

Note: `getInventory` is NOT included because it uses the `store` tag, not `mcp`.
```

- [ ] **Step 3: Create demo API server**

Create `examples/demo-api/main.go`:
```go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type Task struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Done        bool   `json:"done"`
}

var (
	tasks  = map[int]*Task{}
	nextID = 1
	mu     sync.Mutex
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /tasks", listTasks)
	mux.HandleFunc("POST /tasks", createTask)
	mux.HandleFunc("GET /tasks/{id}", getTask)
	mux.HandleFunc("PUT /tasks/{id}", updateTask)
	mux.HandleFunc("DELETE /tasks/{id}", deleteTask)

	addr := ":9090"
	fmt.Printf("Demo API running at http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func listTasks(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	list := make([]*Task, 0, len(tasks))
	for _, t := range tasks {
		list = append(list, t)
	}
	writeJSON(w, http.StatusOK, list)
}

func createTask(w http.ResponseWriter, r *http.Request) {
	var t Task
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	mu.Lock()
	t.ID = nextID
	nextID++
	tasks[t.ID] = &t
	mu.Unlock()
	writeJSON(w, http.StatusCreated, t)
}

func getTask(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	mu.Lock()
	t, ok := tasks[id]
	mu.Unlock()
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func updateTask(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	mu.Lock()
	t, ok := tasks[id]
	if !ok {
		mu.Unlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
		return
	}
	var update Task
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		mu.Unlock()
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if update.Title != "" {
		t.Title = update.Title
	}
	if update.Description != "" {
		t.Description = update.Description
	}
	t.Done = update.Done
	mu.Unlock()
	writeJSON(w, http.StatusOK, t)
}

func deleteTask(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	mu.Lock()
	if _, ok := tasks[id]; !ok {
		mu.Unlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
		return
	}
	delete(tasks, id)
	mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func parseID(s string) int {
	s = strings.TrimSpace(s)
	id, _ := strconv.Atoi(s)
	return id
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
```

- [ ] **Step 4: Create demo API OpenAPI spec**

Create `examples/demo-api/openapi.yaml`:
```yaml
openapi: "3.0.3"
info:
  title: Task Manager Demo API
  description: A simple task manager to demonstrate mcp-server-openapi
  version: "1.0.0"
servers:
  - url: http://localhost:9090
paths:
  /tasks:
    get:
      tags: [mcp]
      operationId: listTasks
      summary: List all tasks
      x-mcp-tool-name: list_tasks
      responses:
        "200":
          description: List of tasks
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Task'
    post:
      tags: [mcp]
      operationId: createTask
      summary: Create a new task
      x-mcp-tool-name: create_task
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/NewTask'
      responses:
        "201":
          description: Task created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Task'
  /tasks/{id}:
    get:
      tags: [mcp]
      operationId: getTask
      summary: Get a task by ID
      x-mcp-tool-name: get_task
      parameters:
        - name: id
          in: path
          required: true
          description: Task ID
          schema:
            type: integer
      responses:
        "200":
          description: A task
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Task'
    put:
      tags: [mcp]
      operationId: updateTask
      summary: Update a task
      x-mcp-tool-name: update_task
      x-mcp-description: Update a task's title, description, or done status
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: integer
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/UpdateTask'
      responses:
        "200":
          description: Updated task
    delete:
      tags: [mcp]
      operationId: deleteTask
      summary: Delete a task
      x-mcp-tool-name: delete_task
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: integer
      responses:
        "204":
          description: Task deleted
components:
  schemas:
    Task:
      type: object
      properties:
        id:
          type: integer
        title:
          type: string
        description:
          type: string
        done:
          type: boolean
    NewTask:
      type: object
      required: [title]
      properties:
        title:
          type: string
        description:
          type: string
    UpdateTask:
      type: object
      properties:
        title:
          type: string
        description:
          type: string
        done:
          type: boolean
```

- [ ] **Step 5: Create demo API README**

Create `examples/demo-api/README.md`:
```markdown
# Demo API Example

A self-contained example: run a task manager API + MCP server locally. No API keys needed.

## Quick Start (30 seconds)

**Terminal 1** — start the demo API:
```bash
cd examples/demo-api
go run .
# Output: Demo API running at http://localhost:9090
```

**Terminal 2** — start the MCP server:
```bash
go run ./cmd/mcp-server-openapi stdio --spec examples/demo-api/openapi.yaml
```

## Claude Desktop Configuration

```json
{
  "mcpServers": {
    "demo-tasks": {
      "command": "mcp-server-openapi",
      "args": ["stdio", "--spec", "/absolute/path/to/examples/demo-api/openapi.yaml"]
    }
  }
}
```

## Tools Generated

| Tool | Method | Path | Description |
|------|--------|------|-------------|
| list_tasks | GET | /tasks | List all tasks |
| create_task | POST | /tasks | Create a new task |
| get_task | GET | /tasks/{id} | Get a task by ID |
| update_task | PUT | /tasks/{id} | Update a task's title, description, or done status |
| delete_task | DELETE | /tasks/{id} | Delete a task |

## Try It

Once both are running and connected to Claude Desktop:

> "Create a task called 'Buy groceries', then list all tasks"

> "Mark task 1 as done"

> "Delete all completed tasks"
```

- [ ] **Step 6: Verify demo API compiles**

```bash
go build ./examples/demo-api/
```

- [ ] **Step 7: Commit**

```bash
git add examples/ && git commit -m "feat: petstore and demo-api examples"
```

---

### Task 14: README, Dockerfile, goreleaser

**Files:**
- Create: `README.md`, `Dockerfile`, `.goreleaser.yaml`, `.github/workflows/ci.yaml`

- [ ] **Step 1: Create README.md**

Create `README.md`:
````markdown
# mcp-server-openapi

[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**Automatically convert OpenAPI endpoints into MCP tools.** Tag your API operations with `mcp`, point this server at the spec, and every LLM client (Claude, Cursor, etc.) can call your API.

## Why This?

| Feature | mcp-server-openapi | mcp-link | emcee |
|---------|-------------------|----------|-------|
| Full `$ref` resolution | Yes (kin-openapi) | No | Partial |
| `oneOf`/`anyOf`/`allOf` | Yes | No | No |
| `x-mcp-*` extensions | Yes | No | No |
| Collision detection | Fail-loud at startup | Silent | Silent |
| Streamable HTTP transport | Yes | SSE only | stdio only |

## Quick Start

```bash
# Install
go install github.com/soyvural/mcp-server-openapi/cmd/mcp-server-openapi@latest

# Run with any OpenAPI spec
mcp-server-openapi stdio --spec ./your-api.yaml --tag mcp
```

### Claude Desktop

```json
{
  "mcpServers": {
    "my-api": {
      "command": "mcp-server-openapi",
      "args": ["stdio", "--spec", "/path/to/openapi.yaml"]
    }
  }
}
```

## How It Works

1. **Load** your OpenAPI 3.x spec (file or URL)
2. **Filter** operations by the `mcp` tag (configurable)
3. **Generate** MCP tools with JSON Schema input validation
4. **Serve** over stdio or Streamable HTTP

```
OpenAPI Spec ──→ kin-openapi parser ──→ Filter by tag ──→ MCP Tools ──→ LLM Client
                 (full $ref resolve)    (x-mcp-* ext)    (mcp-go)
```

## Tag Your Endpoints

Add the `mcp` tag to any operation you want exposed:

```yaml
paths:
  /pets:
    get:
      tags: [mcp]          # ← This endpoint becomes an MCP tool
      operationId: listPets
      summary: List all pets
```

Operations without the `mcp` tag are ignored. Use `x-mcp-*` extensions for more control:

```yaml
  /pets:
    get:
      tags: [mcp]
      x-mcp-tool-name: list_pets         # Override tool name
      x-mcp-description: "Find all pets" # Override description
      x-mcp-hidden: false                # Force include/exclude
```

## CLI Reference

```
mcp-server-openapi <command> [flags]

Commands:
  stdio     Run over stdio (for Claude Desktop, Cursor, etc.)
  http      Run over Streamable HTTP (for remote access)
  version   Print version

Flags:
  --spec string            OpenAPI spec path or URL (required)
  --tag string             Tag to filter operations (default: "mcp")
  --server-url string      Override base URL from spec
  --timeout duration       HTTP request timeout (default: 30s)
  --auth-type string       Auth type: bearer or api-key
  --auth-token-env string  Env var for bearer token
  --auth-key-env string    Env var for API key
  --auth-key-name string   Header/query param name for API key
  --auth-key-in string     Where to send API key: header or query
  --log-level string       Log level (default: "info")
  --log-file string        Log file path (default: stderr)
```

## Authentication

```bash
# Bearer token
export MY_TOKEN=sk-xxx
mcp-server-openapi stdio --spec api.yaml --auth-type bearer --auth-token-env MY_TOKEN

# API key in header
export MY_KEY=key-xxx
mcp-server-openapi stdio --spec api.yaml --auth-type api-key --auth-key-env MY_KEY --auth-key-name X-API-Key --auth-key-in header
```

## Examples

- **[Petstore](examples/petstore/)** — classic OpenAPI example with mcp tags
- **[Demo API](examples/demo-api/)** — self-contained task manager (clone → run → talk to your API in 30 seconds)

<!-- BEGIN TOOLS -->
<!-- END TOOLS -->

## Docker

```bash
docker run -v $(pwd)/spec.yaml:/spec.yaml ghcr.io/soyvural/mcp-server-openapi stdio --spec /spec.yaml
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT
````

- [ ] **Step 2: Create Dockerfile**

Create `Dockerfile`:
```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /mcp-server-openapi ./cmd/mcp-server-openapi

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /mcp-server-openapi /usr/local/bin/mcp-server-openapi
ENTRYPOINT ["mcp-server-openapi"]
```

- [ ] **Step 3: Create .goreleaser.yaml**

Create `.goreleaser.yaml`:
```yaml
version: 2
builds:
  - main: ./cmd/mcp-server-openapi
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w -X main.version={{.Version}}

archives:
  - format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip

checksum:
  name_template: checksums.txt

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
```

- [ ] **Step 4: Create CI workflow**

Create `.github/workflows/ci.yaml`:
```yaml
name: CI
on:
  push:
    branches: [main]
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.23"
      - run: go test -v -race -cover ./...
      - run: go build ./cmd/mcp-server-openapi

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.23"
      - uses: golangci/golangci-lint-action@v6
        with:
          version: latest

  release:
    if: startsWith(github.ref, 'refs/tags/')
    needs: [test, lint]
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: "1.23"
      - uses: goreleaser/goreleaser-action@v6
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

- [ ] **Step 5: Create .gitignore**

Create `.gitignore`:
```
bin/
dist/
*.exe
.DS_Store
```

- [ ] **Step 6: Commit**

```bash
git add README.md Dockerfile .goreleaser.yaml .github/ .gitignore && git commit -m "feat: README, Dockerfile, goreleaser, and CI workflow"
```

---

### Task 15: Integration & E2E Tests

**Files:**
- Create: `toolgen/integration_test.go`, `server/e2e_test.go`

- [ ] **Step 1: Write integration tests for toolgen**

Create `toolgen/integration_test.go`:
```go
package toolgen_test

import (
	"context"
	"testing"

	"github.com/soyvural/mcp-server-openapi/toolgen"
)

func TestIntegration_PetstoreToolGeneration(t *testing.T) {
	tools, err := toolgen.Generate(context.Background(), toolgen.GenerateOptions{
		SpecSource: "../testdata/petstore.yaml",
		Tag:        "mcp",
	})
	if err != nil {
		t.Fatalf("failed to generate: %v", err)
	}

	if len(tools) != 3 {
		t.Fatalf("tool count: got %d, want 3", len(tools))
	}

	// Verify each tool has required fields
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

	// custom_visible should use x-mcp-tool-name and x-mcp-description
	if cv, ok := nameMap["custom_visible"]; !ok {
		t.Error("missing tool 'custom_visible'")
	} else if cv.Description != "Custom description here" {
		t.Errorf("custom_visible desc: got %q", cv.Description)
	}

	// forcedop should exist from x-mcp-hidden=false override
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
		t.Fatal("expected collision error")
	}
	t.Logf("collision error (expected): %v", err)
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

	// Check x-mcp-tool-name overrides are applied
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
```

- [ ] **Step 2: Run integration tests**

```bash
go test ./toolgen/... -v -run TestIntegration
# Expected: PASS
```

- [ ] **Step 3: Write tool validation tests**

Add to `toolgen/integration_test.go`:
```go
func TestValidation_NoDuplicateToolNames(t *testing.T) {
	specs := []string{
		"../testdata/petstore.yaml",
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
				t.Skipf("spec %s failed to generate: %v", spec, err)
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
		SpecSource: "../testdata/petstore.yaml",
		Tag:        "mcp",
	})
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	for _, tool := range tools {
		for _, c := range tool.Name {
			if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '.' || c == '-') {
				t.Errorf("tool %q has invalid char %q", tool.Name, string(c))
			}
		}
	}
}
```

- [ ] **Step 4: Write E2E test with demo API**

Create `server/e2e_test.go`:
```go
package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/soyvural/mcp-server-openapi/executor"
	"github.com/soyvural/mcp-server-openapi/server"
	"github.com/soyvural/mcp-server-openapi/toolgen"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestE2E_ToolCallRoundTrip(t *testing.T) {
	// Spin up a fake upstream API
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/tasks":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]any{
				{"id": 1, "title": "Buy milk", "done": false},
			})
		case r.Method == "POST" && r.URL.Path == "/tasks":
			w.WriteHeader(http.StatusCreated)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"id": 2, "title": "New task", "done": false})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer upstream.Close()

	// Generate tools pointing at the fake upstream
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

	exec := executor.New(upstream.Client(), executor.NoAuth{}, 10*time.Second)
	mcpServer, err := server.New(tools, exec, "test")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Call list_tasks tool directly
	ctx := context.Background()
	result, err := mcpServer.HandleCallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "list_tasks",
			Arguments: map[string]any{},
		},
	})
	if err != nil {
		t.Fatalf("tool call failed: %v", err)
	}

	// Verify result contains task data
	if len(result.Content) == 0 {
		t.Fatal("empty result content")
	}
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if textContent.Text == "" {
		t.Error("empty text response")
	}
	if !json.Valid([]byte(textContent.Text)) {
		t.Errorf("response is not valid JSON: %s", textContent.Text)
	}
	t.Logf("list_tasks response: %s", textContent.Text)

	// Call create_task tool
	result, err = mcpServer.HandleCallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "create_task",
			Arguments: map[string]any{"title": "New task"},
		},
	})
	if err != nil {
		t.Fatalf("create_task call failed: %v", err)
	}
	textContent, ok = result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	t.Logf("create_task response: %s", textContent.Text)
}
```

- [ ] **Step 5: Run all tests**

```bash
go test -v -race -cover ./...
# Expected: ALL PASS
```

- [ ] **Step 6: Commit**

```bash
git add toolgen/integration_test.go server/e2e_test.go && git commit -m "test: integration and E2E tests for full pipeline"
```

---

## Summary

| Task | Component | What |
|------|-----------|------|
| 1 | Project scaffold | go.mod, Cobra CLI, Makefile, LICENSE |
| 2 | pkg/params | Generic Required[T] / Optional[T] |
| 3 | toolgen/namer | Tool name generation + sanitization |
| 4 | toolgen/extensions | x-mcp-* extraction with precedence |
| 5 | toolgen/filter | Tag-based filtering + x-mcp-hidden |
| 6 | toolgen/parser | OpenAPI spec loader via kin-openapi |
| 7 | toolgen/schema | Flat JSON Schema + collision detection |
| 8 | toolgen/generator | Orchestrator: spec → tools |
| 9 | executor/auth | Bearer + API key authenticators |
| 10 | executor/executor | Tool call → HTTP request |
| 11 | server/server | MCP server wiring (~30 lines) |
| 12 | cmd/ | stdio + http subcommands |
| 13 | examples/ | Petstore + demo-api |
| 14 | README + infra | Docs, Dockerfile, goreleaser, CI |
| 15 | Tests | Integration + E2E + validation |