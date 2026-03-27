# Demo API Example - Task Manager

A self-contained Go HTTP server that demonstrates MCP server integration with a real, working API. Get up and running in 30 seconds.

## 30-Second Quick Start

**Terminal 1 - Start the API server:**

```bash
cd examples/demo-api
go run main.go
```

You should see:
```
Task Manager API starting on :9090
```

**Terminal 2 - Start the MCP server:**

```bash
mcp-server-openapi --spec examples/demo-api/openapi.yaml
```

That's it! The demo API is now running on `http://localhost:9090` and exposed as MCP tools.

## Configure Claude Desktop

Edit your Claude Desktop configuration file:

- **macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows:** `%APPDATA%\Claude\claude_desktop_config.json`

Add the following configuration:

```json
{
  "mcpServers": {
    "task-manager": {
      "command": "/path/to/mcp-server-openapi",
      "args": [
        "--spec",
        "/path/to/examples/demo-api/openapi.yaml"
      ]
    }
  }
}
```

Restart Claude Desktop to load the tools.

## Available Tools

All 5 task management operations are exposed with clean, semantic names using `x-mcp-tool-name`:

| Tool Name | Method | Endpoint | Description |
|-----------|--------|----------|-------------|
| `list_tasks` | GET | `/tasks` | Retrieve all tasks from the task manager |
| `create_task` | POST | `/tasks` | Add a new task to the task manager |
| `get_task` | GET | `/tasks/{id}` | Retrieve a specific task by its ID |
| `update_task` | PUT | `/tasks/{id}` | Update an existing task with new information |
| `delete_task` | DELETE | `/tasks/{id}` | Remove a task from the task manager |

## Example Prompts

Try these prompts in Claude Desktop after configuration:

- "Show me all tasks in the task manager"
- "Create a new task: Write documentation for the API"
- "Get task 1"
- "Update task 1 to mark it as completed"
- "Delete task 1"
- "Create three tasks: buy groceries, walk the dog, and finish project proposal"

## API Details

**Server:** The Go server runs on `http://localhost:9090`

**Storage:** In-memory with thread-safe access using `sync.Mutex`

**Sample Data:** The server starts with one pre-created task (ID: 1) to demonstrate the API immediately.

**Go Version:** Requires Go 1.22+ for the new `net/http` routing patterns (`mux.HandleFunc("GET /tasks", ...)`)

## Testing the API Directly

You can test the API with curl:

```bash
# List all tasks
curl http://localhost:9090/tasks

# Create a new task
curl -X POST http://localhost:9090/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"Test task","description":"A test task","completed":false}'

# Get a specific task
curl http://localhost:9090/tasks/1

# Update a task
curl -X PUT http://localhost:9090/tasks/1 \
  -H "Content-Type: application/json" \
  -d '{"title":"Updated task","description":"Updated description","completed":true}'

# Delete a task
curl -X DELETE http://localhost:9090/tasks/1
```

## MCP Extensions Used

This example demonstrates all MCP-specific OpenAPI extensions:

- **`mcp` tag:** Marks endpoints to be exposed as tools (all 5 operations have this tag)
- **`x-mcp-tool-name`:** Provides clean, semantic tool names (`list_tasks` instead of `listTasks`)
- **`x-mcp-description`:** Custom descriptions optimized for LLM understanding

## Building the Server

To compile the server as a standalone binary:

```bash
go build -o task-manager ./examples/demo-api/
./task-manager
```

## Architecture

**~120 lines of idiomatic Go code:**

- `Task` struct: JSON-serializable task representation
- `TaskStore`: Thread-safe in-memory storage with `sync.Mutex`
- HTTP handlers using Go 1.22+ routing patterns
- Full CRUD operations with proper status codes
- No external dependencies beyond standard library

Perfect for testing MCP server functionality without external API dependencies or authentication complexity.
