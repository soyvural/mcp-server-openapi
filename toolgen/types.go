package toolgen

import "encoding/json"

// GeneratedTool holds MCP tool definition and request metadata.
type GeneratedTool struct {
	Name        string
	Description string
	InputSchema json.RawMessage
	ParamMeta   []ParamMeta
	Path        string
	Method      string
	ServerURL   string
}
