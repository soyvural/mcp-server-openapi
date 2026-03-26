package toolgen

// MCPExtensions holds x-mcp-* extension fields.
type MCPExtensions struct {
	ToolName    string
	Description string
	Hidden      *bool
}

// ExtractExtensions extracts x-mcp-* fields, preferring operation over path level.
func ExtractExtensions(opExt, pathExt map[string]any) MCPExtensions {
	var ext MCPExtensions
	if v, ok := opExt["x-mcp-tool-name"].(string); ok {
		ext.ToolName = v
	}
	if v, ok := opExt["x-mcp-description"].(string); ok {
		ext.Description = v
	}
	if v, ok := opExt["x-mcp-hidden"].(bool); ok {
		ext.Hidden = &v
	} else if v, ok := pathExt["x-mcp-hidden"].(bool); ok {
		ext.Hidden = &v
	}
	return ext
}
