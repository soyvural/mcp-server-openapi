package toolgen

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/getkin/kin-openapi/openapi3"
)

// SchemaResult holds converted JSON Schema and parameter location metadata.
type SchemaResult struct {
	InputSchema json.RawMessage
	ParamMeta   []ParamMeta
}

// ParamMeta tracks parameter source for request reconstruction.
type ParamMeta struct {
	Name     string
	In       string // "path", "query", "header", "body"
	Required bool
}

// ConvertSchema merges parameters and body into flat JSON Schema, erroring on name collisions.
func ConvertSchema(params openapi3.Parameters, body *openapi3.RequestBodyRef) (*SchemaResult, error) {
	properties := make(map[string]map[string]any)
	required := []string{}
	var meta []ParamMeta

	for _, pr := range params {
		if pr == nil || pr.Value == nil {
			continue
		}
		p := pr.Value
		pm := ParamMeta{
			Name:     p.Name,
			In:       p.In,
			Required: p.Required,
		}
		meta = append(meta, pm)

		prop := make(map[string]any)
		if p.Schema != nil && p.Schema.Value != nil {
			prop = schemaToMap(p.Schema)
		}
		properties[p.Name] = prop

		if p.Required {
			required = append(required, p.Name)
		}
	}

	if body != nil && body.Value != nil && body.Value.Content != nil {
		mt := body.Value.Content.Get("application/json")
		if mt != nil && mt.Schema != nil && mt.Schema.Value != nil {
			bodySchema := mt.Schema.Value

			for name, propRef := range bodySchema.Properties {
				if _, exists := properties[name]; exists {
					return nil, fmt.Errorf("parameter name collision: %q exists in both parameters and request body", name)
				}

				prop := make(map[string]any)
				if propRef != nil {
					prop = schemaToMap(propRef)
				}
				properties[name] = prop

				meta = append(meta, ParamMeta{
					Name:     name,
					In:       "body",
					Required: containsString(bodySchema.Required, name),
				})
			}

			for _, r := range bodySchema.Required {
				if containsString(required, r) {
					continue
				}
				required = append(required, r)
			}
		}
	}

	sort.Strings(required)

	schema := map[string]any{
		"type": "object",
	}
	if len(properties) > 0 {
		schema["properties"] = properties
	}
	if len(required) > 0 {
		schema["required"] = required
	}

	raw, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}

	return &SchemaResult{
		InputSchema: raw,
		ParamMeta:   meta,
	}, nil
}

// schemaToMap converts a kin-openapi SchemaRef to a JSON Schema map representation.
func schemaToMap(ref *openapi3.SchemaRef) map[string]any {
	if ref == nil || ref.Value == nil {
		return map[string]any{}
	}
	s := ref.Value
	m := make(map[string]any)

	if s.Type != nil && len(*s.Type) > 0 {
		types := *s.Type
		if len(types) == 1 {
			m["type"] = types[0]
		} else {
			m["type"] = []string(types)
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
	if s.Min != nil {
		m["minimum"] = *s.Min
	}
	if s.Max != nil {
		m["maximum"] = *s.Max
	}
	if s.MinLength != 0 {
		m["minLength"] = s.MinLength
	}
	if s.MaxLength != nil {
		m["maxLength"] = *s.MaxLength
	}

	if len(s.Properties) > 0 {
		props := make(map[string]any)
		for name, propRef := range s.Properties {
			props[name] = schemaToMap(propRef)
		}
		m["properties"] = props
	}
	if len(s.Required) > 0 {
		m["required"] = s.Required
	}

	if s.Items != nil {
		m["items"] = schemaToMap(s.Items)
	}

	if len(s.OneOf) > 0 {
		var variants []map[string]any
		for _, v := range s.OneOf {
			variants = append(variants, schemaToMap(v))
		}
		m["oneOf"] = variants
	}
	if len(s.AnyOf) > 0 {
		var variants []map[string]any
		for _, v := range s.AnyOf {
			variants = append(variants, schemaToMap(v))
		}
		m["anyOf"] = variants
	}
	if len(s.AllOf) > 0 {
		var variants []map[string]any
		for _, v := range s.AllOf {
			variants = append(variants, schemaToMap(v))
		}
		m["allOf"] = variants
	}

	return m
}

func containsString(ss []string, target string) bool {
	for _, s := range ss {
		if s == target {
			return true
		}
	}
	return false
}
