package toolgen

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
)

// GenerateOptions configures tool generation.
type GenerateOptions struct {
	SpecSource string
	Tag        string
	ServerURL  string
}

// Generate produces MCP tools from an OpenAPI spec.
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
	nameSet := make(map[string]string)

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
				slog.Debug("skipping operation", "path", path, "method", method)
				continue
			}

			ext := ExtractExtensions(op.Extensions, pathExt)

			name := GenerateToolName(path, method, op.OperationID, ext.ToolName)
			if existing, dup := nameSet[name]; dup {
				return nil, fmt.Errorf("duplicate tool name %q: operations %q and %q", name, existing, op.OperationID)
			}
			nameSet[name] = op.OperationID

			desc := op.Summary
			if ext.Description != "" {
				desc = ext.Description
			} else if op.Description != "" && desc == "" {
				desc = op.Description
			}

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

	slog.Info("tool generation complete", "tools_generated", len(tools))
	return tools, nil
}
