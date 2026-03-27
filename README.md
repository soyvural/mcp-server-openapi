# mcp-server-openapi

[![CI](https://github.com/soyvural/mcp-server-openapi/actions/workflows/ci.yaml/badge.svg)](https://github.com/soyvural/mcp-server-openapi/actions/workflows/ci.yaml)
[![Go Version](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go&logoColor=white)](https://golang.org/doc/go1.23)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![GoDoc](https://pkg.go.dev/badge/github.com/soyvural/mcp-server-openapi.svg)](https://pkg.go.dev/github.com/soyvural/mcp-server-openapi)
[![MCP](https://img.shields.io/badge/MCP-Compatible-8A2BE2?logo=data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSIyNCIgaGVpZ2h0PSIyNCIgdmlld0JveD0iMCAwIDI0IDI0IiBmaWxsPSJ3aGl0ZSI+PGNpcmNsZSBjeD0iMTIiIGN5PSIxMiIgcj0iMTAiLz48L3N2Zz4=)](https://modelcontextprotocol.io)
[![OpenAPI](https://img.shields.io/badge/OpenAPI-3.x-6BA539?logo=openapiinitiative&logoColor=white)](https://www.openapis.org/)
[![Go Report Card](https://goreportcard.com/badge/github.com/soyvural/mcp-server-openapi)](https://goreportcard.com/report/github.com/soyvural/mcp-server-openapi)

**Automatically convert OpenAPI endpoints into MCP tools**

Turn any OpenAPI-documented API into a set of tools that Claude Desktop (or any MCP client) can call. Just tag the endpoints you want to expose, point the server at your spec, and you're done.

---

## Key Strengths

- **Battle-tested OpenAPI parsing** — Built on [kin-openapi](https://github.com/getkin/kin-openapi), the most widely used Go OpenAPI library, with full `$ref` resolution out of the box
- **Fine-grained control** — Tag-based filtering (`mcp` tag), `x-mcp-hidden` to explicitly show/hide operations, custom tool names (`x-mcp-tool-name`), and custom descriptions (`x-mcp-description`)
- **Collision-safe** — Automatic tool name collision detection with operationId + path-based fallback ensures no two tools silently overwrite each other
- **Flexible authentication** — Bearer tokens and API keys (header or query) with environment variable-based credential management
- **Zero config for simple cases** — Point at a spec, tag your endpoints, done. No code generation, no boilerplate

---

## Quick Start

### 1. Install

```bash
go install github.com/soyvural/mcp-server-openapi/cmd/mcp-server-openapi@latest
```

Or clone and build:

```bash
git clone https://github.com/soyvural/mcp-server-openapi.git
cd mcp-server-openapi
make build
```

### 2. Run the Server

```bash
mcp-server-openapi --spec ./examples/weather/weather.yaml
```

### 3. Configure Claude Desktop

Edit your Claude Desktop config:

- **macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows:** `%APPDATA%\Claude\claude_desktop_config.json`

Add:

```json
{
  "mcpServers": {
    "openapi": {
      "command": "/path/to/mcp-server-openapi",
      "args": [
        "--spec",
        "/path/to/your/openapi.yaml"
      ]
    }
  }
}
```

Restart Claude Desktop. You should now see the exposed tools in the MCP tools panel.

---

## How It Works

```
┌──────────────────┐
│  OpenAPI Spec    │  (YAML or JSON, local or URL)
│  (tagged with    │
│   "mcp")         │
└────────┬─────────┘
         │
         v
┌────────────────────────────────────────────────────┐
│  mcp-server-openapi                                │
│                                                    │
│  1. Parse spec (kin-openapi)                       │
│  2. Filter operations by tag                       │
│  3. Generate MCP tool schema (JSON Schema)         │
│  4. Serve tools via stdio (mcp-go SDK)             │
│  5. Execute HTTP requests when called              │
└────────┬───────────────────────────────────────────┘
         │
         v
┌──────────────────┐
│  Claude Desktop  │  (or any MCP client)
│  calls tools     │
└──────────────────┘
         │
         v
┌──────────────────┐
│  Your API        │  (GET /items/123, POST /orders, etc.)
└──────────────────┘
```

**Key steps:**
1. Operations with the `mcp` tag (configurable via `--tag`) are selected
2. Each operation becomes an MCP tool with a JSON Schema input definition
3. Path params, query params, headers, and request body are mapped to tool arguments
4. When Claude calls a tool, we build and execute the HTTP request
5. Response is returned as text (JSON, XML, plain text, etc.)

---

## Tag Your Endpoints

### Basic Tagging

Add the `mcp` tag to any operation you want to expose:

```yaml
paths:
  /items/{id}:
    get:
      tags:
        - mcp
      operationId: getItemById
      summary: Get an item by ID
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: integer
      responses:
        "200":
          description: OK
```

By default, only operations with the `mcp` tag are exposed. Operations without the tag are ignored.

### OpenAPI Extensions (x-mcp-*)

Fine-tune tool generation with custom extensions:

```yaml
paths:
  /users/{id}:
    get:
      tags: [mcp]
      operationId: getUserById
      summary: Retrieve user by ID
      x-mcp-tool-name: get_user         # Override default tool name
      x-mcp-description: |               # Custom description for Claude
        Fetch detailed user information including profile, settings, and metadata.
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        "200":
          description: OK

  /internal/health:
    get:
      tags: [mcp]                        # Tagged, but...
      operationId: healthCheck
      x-mcp-hidden: true                 # Explicitly hidden
      responses:
        "200":
          description: OK

  /debug/stats:
    get:
      tags: [internal]                   # No mcp tag, but...
      operationId: getStats
      x-mcp-hidden: false                # Force visible
      responses:
        "200":
          description: OK
```

**Supported extensions:**
- `x-mcp-tool-name` (string): Override the tool name (defaults to operationId or generated from method+path)
- `x-mcp-description` (string): Override the tool description (defaults to summary or description)
- `x-mcp-hidden` (boolean): Explicitly hide (true) or show (false) the operation, regardless of tag

---

## CLI Reference

```
mcp-server-openapi [flags]

Flags:
  --spec string             OpenAPI spec file path or URL (required)
  --tag string              Tag to filter operations (default: "mcp")
  --server-url string       Override base URL from spec
  --timeout duration        HTTP request timeout (default: 30s)
  --auth-type string        Authentication type: bearer or api-key
  --auth-token-env string   Env var name for bearer token (e.g., GITHUB_TOKEN)
  --auth-key-env string     Env var name for API key (e.g., API_KEY)
  --auth-key-name string    Header/query param name for API key (e.g., X-API-Key)
  --auth-key-in string      Where to send API key: header or query
  --log-level string        Log level: debug, info, warn, error (default: info)
  --log-file string         Log file path (default: stderr)

Commands:
  version                   Print version
```

**Environment variables:**

All flags can be set via `OPENAPI_MCP_*` env vars (e.g., `OPENAPI_MCP_SPEC`, `OPENAPI_MCP_TAG`).

---

## Authentication

### Bearer Token

```bash
export GITHUB_TOKEN="ghp_..."
mcp-server-openapi \
  --spec https://api.github.com/openapi.yaml \
  --auth-type bearer \
  --auth-token-env GITHUB_TOKEN
```

The token is read from the specified env var and sent as `Authorization: Bearer <token>`.

### API Key (Header)

```bash
export MY_API_KEY="sk_..."
mcp-server-openapi \
  --spec ./api.yaml \
  --auth-type api-key \
  --auth-key-env MY_API_KEY \
  --auth-key-name X-API-Key \
  --auth-key-in header
```

Sends `X-API-Key: sk_...` with every request.

### API Key (Query Parameter)

```bash
export MY_API_KEY="abc123"
mcp-server-openapi \
  --spec ./api.yaml \
  --auth-type api-key \
  --auth-key-env MY_API_KEY \
  --auth-key-name api_key \
  --auth-key-in query
```

Appends `?api_key=abc123` to every request URL.

---

## Examples

### Weather (Public API)

The [Open-Meteo](https://open-meteo.com/) Weather API — free, no API key required:

```bash
mcp-server-openapi --spec examples/weather/weather.yaml
```

See [examples/weather/README.md](examples/weather/README.md) for details.

### Demo API (Full Feature Showcase)

A synthetic API demonstrating all parameter types, auth, and extensions:

```bash
cd examples/demo-api
go run main.go  # Start local test server on :8080
```

In another terminal:

```bash
mcp-server-openapi --spec http://localhost:8080/openapi.yaml
```

See [examples/demo-api/main.go](examples/demo-api/main.go) for the full implementation.

---

## Docker

Build the image:

```bash
docker build -t mcp-server-openapi .
```

Run with a local spec:

```bash
docker run --rm -i \
  -v $(pwd)/examples:/specs \
  mcp-server-openapi --spec /specs/weather/weather.yaml
```

Run with authentication:

```bash
docker run --rm -i \
  -e GITHUB_TOKEN="ghp_..." \
  mcp-server-openapi \
    --spec https://api.github.com/openapi.yaml \
    --auth-type bearer \
    --auth-token-env GITHUB_TOKEN
```

---

## Contributing

Contributions welcome! Please open an issue or PR.

**Development:**

```bash
make test           # Run tests
make lint           # Run golangci-lint
make build          # Build binary to ./bin/
```

---

## License

MIT License - see [LICENSE](LICENSE) for details.
