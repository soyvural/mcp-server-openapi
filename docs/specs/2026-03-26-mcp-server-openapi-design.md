# mcp-server-openapi Design Spec

**Date**: 2026-03-26
**Status**: Approved
**Language**: Go

## Problem

API authors want to expose their REST APIs as MCP tools so that LLMs (Claude, GPT, etc.) can call them. Today they must either handcraft an MCP server or use existing tools (mcp-link, emcee) that have buggy schema handling, no `$ref` resolution, and no support for `oneOf/anyOf/allOf`.

## Solution

A Go CLI that reads an OpenAPI 3.x spec, filters operations by the `mcp` tag (or `x-mcp-*` extensions), and serves them as MCP tools over stdio or Streamable HTTP. Uses `kin-openapi` for correct schema resolution and `mcp-go` for protocol compliance.

## Differentiators

1. **Correct schema handling** — full `$ref` resolution, `oneOf/anyOf/allOf`, circular ref detection via kin-openapi. No existing Go project does this.
2. **`x-mcp-*` vendor extension standard** — a reusable extension spec that API authors can adopt.
3. **Fail-loud on collision** — no silent schema mutations. Startup fails with a clear error if param names collide.

## Architecture

```
cmd/mcp-server-openapi/     Composition root (Cobra CLI, wiring)
  main.go                    Root command, version
  stdio.go                   stdio subcommand
  http.go                    Streamable HTTP subcommand

toolgen/                     Core domain: OpenAPI -> MCP tool definitions
  generator.go               Orchestrator: loads spec, filters, converts
  parser.go                  Loads OpenAPI spec from file/URL via kin-openapi
  filter.go                  Tag-based + x-mcp-hidden filtering
  schema.go                  Params + body -> flat JSON Schema, collision detection
  namer.go                   operationId -> sanitized tool name
  extensions.go              x-mcp-* extraction and precedence

executor/                    Infrastructure: executes tool calls as HTTP requests
  executor.go                Decomposes MCP args -> path/query/header/body, makes HTTP call
  auth.go                    Authenticator interface + bearer/api_key implementations
  response.go                HTTP response -> MCP tool result, error mapping

server/                      Application: wires toolgen + executor into mcp-go server
  server.go                  Registers tools, sets up handlers (~30 lines)

pkg/params/                  Shared utilities
  params.go                  Generic RequiredParam[T] / OptionalParam[T] helpers

tools/gendocs/               README documentation generator
  main.go                    Reads tools, generates markdown table

examples/
  petstore/                  Modified petstore.yaml with mcp tags + README
  demo-api/                  Self-contained Go HTTP server + OpenAPI spec
```

### Dependency Rule

Inner layers never import outer layers:
- `toolgen/` depends only on `kin-openapi` and stdlib
- `executor/` depends only on stdlib (+ `net/http`)
- `server/` depends on `toolgen/`, `executor/`, and `mcp-go`
- `cmd/` wires everything together

### Interfaces (2 only)

```go
// executor/auth.go
// Authenticator injects credentials into outbound HTTP requests.
// Multiple implementations: BearerAuth, APIKeyAuth.
type Authenticator interface {
    Apply(req *http.Request) error
}

// executor/executor.go
// RequestExecutor executes HTTP requests for tool invocations.
// Interface exists for testability.
type RequestExecutor interface {
    Execute(ctx context.Context, req *ToolRequest) (*ToolResponse, error)
}
```

Everything else is concrete types.

## Tool Generation

### Operation Discovery

1. Load spec via `kin-openapi` loader (file path or URL)
2. Validate spec
3. Resolve all `$ref` references (kin-openapi does this automatically)
4. Iterate `doc.Paths.Map()` -> for each path, iterate `.Operations()`
5. Apply filter chain: tag match OR x-mcp-hidden override

### Filtering Logic

```
Include an operation if:
  (operation has "mcp" tag) AND (x-mcp-hidden != true)
  OR
  (x-mcp-hidden == false)  // explicit include regardless of tag
```

Precedence: operation-level x-mcp-hidden > path-level x-mcp-hidden > tag filter.

### Tool Naming

1. If `x-mcp-tool-name` extension exists on operation, use it (sanitized)
2. Else if `operationId` exists, use it (sanitized)
3. Else generate from `method_path`: `GET /pets/{petId}` -> `get_pets_petId`

Sanitization: replace non-alphanumeric chars with `_`, collapse consecutive `_`, trim edges, lowercase. Fail at startup if two tools collide after naming.

### Parameter Mapping (Always Flat)

All parameters (path, query, header) and request body properties are merged into a single flat JSON Schema object.

```
GET /pets/{petId}?verbose=true
->
{
  "type": "object",
  "properties": {
    "petId": {"type": "integer", "description": "ID of pet to return"},
    "verbose": {"type": "boolean", "description": "Include extra details"}
  },
  "required": ["petId"]
}
```

For request bodies with `application/json` content type, the body schema properties are merged at the top level alongside path/query params.

**Collision handling**: If any parameter name collides with a body property name, the server fails at startup with:
```
Error: parameter name collision in operation "updatePet": "name" exists in both query params and request body.
Use x-mcp-tool-name or restructure the operation to resolve.
```

This is a deliberate choice: silent schema mutations are worse than loud failures.

### Schema Conversion

kin-openapi resolves `$ref`, `allOf`, `oneOf`, `anyOf` into concrete `*openapi3.Schema` objects. The converter:

1. Walks the resolved schema
2. Strips OpenAPI-specific fields (`readOnly`, `writeOnly`, `xml`, `externalDocs`)
3. Preserves JSON Schema fields (`type`, `properties`, `required`, `enum`, `oneOf`, `anyOf`, `allOf`, `items`, `format`, `description`, `default`, `minimum`, `maximum`, `pattern`, `minLength`, `maxLength`)
4. Outputs `json.RawMessage` for `mcp.NewToolWithRawSchema()`

### x-mcp-* Extensions (v1)

| Extension | Level | Type | Purpose |
|-----------|-------|------|---------|
| `x-mcp-tool-name` | operation | string | Override generated tool name |
| `x-mcp-description` | operation | string | Override OpenAPI description/summary |
| `x-mcp-hidden` | operation, path | bool | Exclude (true) or force-include (false) |

Future extensions (v1.1+): `x-mcp-group`, `x-mcp-confirm`, `x-mcp-read-only`.

## Request Execution

When a tool is called:

1. Look up the operation metadata (path template, method, param locations)
2. Decompose flat MCP arguments back into path params, query params, headers, and body
3. Substitute path params into URL template: `/pets/{petId}` + `petId=123` -> `/pets/123`
4. Build query string from query params
5. Serialize remaining args as JSON request body
6. Apply `Authenticator` to inject auth headers
7. Execute HTTP request with configured timeout
8. Map response to MCP tool result

### Error Mapping

| HTTP Status | MCP Behavior |
|-------------|-------------|
| 2xx | Return body as text content |
| 400-499 | Return error with status code and body in message |
| 500-599 | Return error with "upstream server error" + status code |
| Timeout | Return error with "request timed out" |
| Connection error | Return error with "failed to connect to upstream" |

All errors include the HTTP status code and response body (truncated to 4KB) for debuggability.

### Authentication

```go
// BearerAuth reads token from env var, adds Authorization: Bearer <token>
type BearerAuth struct{ TokenEnv string }

// APIKeyAuth reads key from env var, adds it as header or query param
type APIKeyAuth struct{
    KeyEnv   string
    In       string // "header" or "query"
    Name     string // header/query param name
}
```

Auth is configured via CLI flags:
```
--auth-type bearer --auth-token-env MY_API_TOKEN
--auth-type api-key --auth-key-env MY_KEY --auth-key-name X-API-Key --auth-key-in header
```

## Transport

- **stdio** (default): `mcp-go` `server.ServeStdio()`. Works with Claude Desktop, Claude Code, Cursor.
- **Streamable HTTP**: `mcp-go` `server.NewStreamableHTTPServer()`. For remote/shared access.

Selected via `--transport stdio|http` flag. HTTP requires `--addr :8080`.

## CLI Interface (Cobra Subcommands)

```
mcp-server-openapi <command> [flags]

Commands:
  stdio       Run MCP server over stdio (default)
  http        Run MCP server over Streamable HTTP
  version     Print version and exit

Persistent Flags (all commands):
  --spec string            OpenAPI spec file path or URL (required)
  --tag string             Tag to filter operations (default: "mcp")
  --server-url string      Override base URL from spec
  --timeout duration       HTTP request timeout (default: 30s)
  --auth-type string       Auth type: bearer or api-key
  --auth-token-env string  Env var for bearer token
  --auth-key-env string    Env var for API key
  --auth-key-name string   Header/query param name for API key
  --auth-key-in string     Where to send API key: header or query
  --log-level string       Log level: debug, info, warn, error (default: "info")
  --log-file string        Log file path (default: stderr)

HTTP-Specific Flags:
  --addr string            Listen address (default: ":8080")
```

Environment variable overrides with `OPENAPI_MCP_` prefix:
- `OPENAPI_MCP_SPEC` → `--spec`
- `OPENAPI_MCP_TAG` → `--tag`
- `OPENAPI_MCP_SERVER_URL` → `--server-url`
- `OPENAPI_MCP_AUTH_TYPE` → `--auth-type`
- `OPENAPI_MCP_AUTH_TOKEN` → `--auth-token-env`

## Logging

Uses Go stdlib `log/slog` with structured JSON output:
- Startup: spec loaded, N operations found, M tools registered
- Per-request: tool name, upstream URL, status code, duration
- Errors: collision details, auth failures, upstream errors

## Examples

### Petstore (`examples/petstore/`)

Modified Petstore 3.0 spec with `mcp` tags on 5 key endpoints:
- `GET /pet/{petId}` (findPetById)
- `GET /pet/findByStatus` (findPetsByStatus)
- `POST /pet` (addPet)
- `PUT /pet` (updatePet)
- `DELETE /pet/{petId}` (deletePet)

README shows Claude Desktop config:
```json
{
  "mcpServers": {
    "petstore": {
      "command": "mcp-server-openapi",
      "args": ["--spec", "/path/to/petstore.yaml"]
    }
  }
}
```

### Demo API (`examples/demo-api/`)

Self-contained Go HTTP server (~150 lines) with:
- `GET /tasks` — list tasks
- `POST /tasks` — create task
- `GET /tasks/{id}` — get task
- `PUT /tasks/{id}` — update task
- `DELETE /tasks/{id}` — delete task

OpenAPI spec with `mcp` tags and `x-mcp-*` extensions. README:
```bash
# Terminal 1: start the demo API
cd examples/demo-api && go run .

# Terminal 2: start the MCP server
mcp-server-openapi --spec examples/demo-api/openapi.yaml --server-url http://localhost:9090
```

## Testing Strategy

- **Unit tests**: table-driven with `cmp.Diff` per package
  - `toolgen/`: spec parsing, filtering, schema conversion, naming, collision detection
  - `executor/`: request building, auth injection, error mapping
- **Integration tests**: load 5+ real-world specs (Petstore, GitHub subset, Stripe subset, a spec with circular refs, a spec with discriminators) and verify tool generation succeeds
- **E2E test**: start demo-api + MCP server, call tools via mcp-go client, verify round-trip

## Build & Release

- `Makefile` with: `build`, `test`, `lint`, `fmt`
- `Dockerfile` for containerized deployment
- `.goreleaser.yaml` for multi-platform binary releases
- GitHub Actions CI: test, lint, build on push; goreleaser on tag

## Patterns Adopted from GitHub & EdgeDelta MCP Servers

### Generic Parameter Extraction (from GitHub MCP Server)

Type-safe helpers to reduce boilerplate in tool handlers:

```go
// pkg/params/params.go
func Required[T any](args map[string]any, key string) (T, error)
func Optional[T any](args map[string]any, key string, defaultVal T) T
```

Used in every tool handler to extract arguments from `mcp.CallToolRequest`.

### Two-Level Error Handling (from GitHub MCP Server)

- **Parameter validation errors** → `mcp.NewToolResultError(msg)` with `nil` error return.
  These are expected client mistakes (missing required param, bad type).
- **Infrastructure errors** → `nil` result with `error` return.
  These are unexpected failures (upstream down, auth expired, network timeout).

```go
func handler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Level 1: param errors → tool result error
    petId, err := params.Required[string](req.GetArguments(), "petId")
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }

    // Level 2: infra errors → error return
    resp, err := executor.Execute(ctx, toolReq)
    if err != nil {
        return nil, fmt.Errorf("upstream request failed: %w", err)
    }
    return mcp.NewToolResultText(resp.Body), nil
}
```

### Context-Based Auth Injection (from EdgeDelta MCP Server)

Auth credentials flow through context, not struct fields. This avoids global state
and is thread-safe for concurrent tool calls:

```go
// Set during server initialization
ctx = context.WithValue(ctx, authKey, authenticator)

// Used in executor
auth := ctx.Value(authKey).(Authenticator)
auth.Apply(req)
```

### Cobra + Viper CLI (from EdgeDelta MCP Server)

Cobra for subcommands (`stdio`, `http`), Viper for env var overrides:

```
mcp-server-openapi stdio --spec ./petstore.yaml --tag mcp
mcp-server-openapi http --spec ./petstore.yaml --addr :8080
```

Environment variables with `OPENAPI_MCP_` prefix override flags:
- `OPENAPI_MCP_SPEC` → `--spec`
- `OPENAPI_MCP_AUTH_TOKEN` → `--auth-token-env`

### Minimal Server Wiring (from EdgeDelta MCP Server)

`server/server.go` stays under 30 lines. Just creates `MCPServer`, registers
generated tools, and returns. No business logic in the wiring layer:

```go
func NewServer(tools []GeneratedTool, executor RequestExecutor) *server.MCPServer {
    s := server.NewMCPServer("mcp-server-openapi", version)
    for _, t := range tools {
        s.AddTool(t.Tool, t.Handler(executor))
    }
    return s
}
```

### Tool Validation Tests (from GitHub MCP Server)

Automated tests that catch regressions in tool definitions:

```go
func TestNoDuplicateToolNames(t *testing.T)      // name collision detection
func TestAllToolsHaveSchema(t *testing.T)         // no empty inputSchema
func TestAllToolsHaveDescription(t *testing.T)    // no missing descriptions
func TestToolNamesAreValid(t *testing.T)          // MCP name regex: [A-Za-z0-9_.-]
```

### Auto-Generated README (from GitHub MCP Server)

A `tools/gendocs/main.go` that reads registered tools and generates a markdown
table in the README between marker comments:

```markdown
<!-- BEGIN TOOLS -->
| Tool | Description | Parameters |
|------|-------------|------------|
| findPetById | Find pet by ID | petId (integer, required) |
<!-- END TOOLS -->
```

Run via `make docs` during CI to keep README in sync with code.

## Dependencies

- `getkin/kin-openapi` — OpenAPI 3.x parsing and validation
- `mark3labs/mcp-go` — MCP protocol server
- `spf13/cobra` — CLI framework with subcommands
- `spf13/viper` — Configuration with env var overrides
- `log/slog` — structured logging (stdlib)
- `net/http` — HTTP client (stdlib)

## Scope Deferred to v1.1

- Config file support (YAML)
- SSE transport
- OAuth2 client_credentials auth
- Basic auth
- `x-mcp-group` extension (toolset grouping)
- `x-mcp-confirm` extension (destructive operation annotations)
- `x-mcp-read-only` extension
- Spec hot-reload
- Rate limiting
- Retry policies
- `multipart/form-data` and `application/x-www-form-urlencoded` content types
