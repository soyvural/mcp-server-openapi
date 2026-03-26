# Petstore Example

This example demonstrates how to expose the Swagger Petstore API as MCP tools. Only pet management operations are exposed (5 tools total) — the store inventory endpoint is intentionally excluded.

## Quick Start

1. **Start the MCP server with the Petstore spec:**

```bash
mcp-server-openapi --spec examples/petstore/petstore.yaml
```

2. **Configure Claude Desktop:**

Edit your Claude Desktop configuration file:

- **macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows:** `%APPDATA%\Claude\claude_desktop_config.json`

Add the following configuration:

```json
{
  "mcpServers": {
    "petstore": {
      "command": "/path/to/mcp-server-openapi",
      "args": [
        "--spec",
        "/path/to/examples/petstore/petstore.yaml"
      ]
    }
  }
}
```

3. **Restart Claude Desktop** and verify the tools are available.

## Available Tools

The following 5 operations are exposed as MCP tools (identified by the `mcp` tag):

| Tool Name | Method | Endpoint | Description |
|-----------|--------|----------|-------------|
| `getPetById` | GET | `/pet/{petId}` | Returns a single pet by ID |
| `deletePet` | DELETE | `/pet/{petId}` | Deletes a pet by ID |
| `findPetsByStatus` | GET | `/pet/findByStatus` | Finds pets by status (available, pending, sold) |
| `addPet` | POST | `/pet` | Add a new pet to the store |
| `updatePet` | PUT | `/pet` | Update an existing pet |

**Note:** The `/store/inventory` endpoint does NOT have the `mcp` tag and will not be exposed as a tool.

## Example Prompts

Try these prompts in Claude Desktop:

- "Find all available pets in the petstore"
- "Get details for pet ID 10"
- "Create a new pet named Max with status available"
- "Update pet ID 10 to status sold"
- "Delete pet ID 5"

## API Authentication

The real Petstore API at `https://petstore3.swagger.io/api/v3` is public and doesn't require authentication. For the `deletePet` operation, you can optionally provide an `api_key` header.

## Schema References

This example uses `$ref` to reference shared schemas (`Pet`, `Category`, `Tag`) defined in the `components/schemas` section, demonstrating proper OpenAPI spec organization.
