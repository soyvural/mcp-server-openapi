package toolgen

import (
	"regexp"
	"strings"
)

var (
	nonAlphanumRe     = regexp.MustCompile(`[^a-z0-9._]`)
	multiUnderscoreRe = regexp.MustCompile(`_+`)
)

// GenerateToolName generates a tool name.
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

// SanitizeToolName converts to valid MCP name: lowercase, alphanumeric, dots, underscores.
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
