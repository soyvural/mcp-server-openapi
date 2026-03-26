package toolgen_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/go-cmp/cmp"
	"github.com/soyvural/mcp-server-openapi/toolgen"
)

func TestConvertSchema_QueryAndPathParams(t *testing.T) {
	params := openapi3.Parameters{
		&openapi3.ParameterRef{
			Value: &openapi3.Parameter{
				Name:     "id",
				In:       "path",
				Required: true,
				Schema: &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: &openapi3.Types{"string"},
					},
				},
			},
		},
		&openapi3.ParameterRef{
			Value: &openapi3.Parameter{
				Name:     "limit",
				In:       "query",
				Required: false,
				Schema: &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: &openapi3.Types{"integer"},
					},
				},
			},
		},
	}

	result, err := toolgen.ConvertSchema(params, nil)
	if err != nil {
		t.Fatalf("TestConvertSchema_QueryAndPathParams: got: %v, want: nil error", err)
	}

	var schema map[string]any
	if err := json.Unmarshal(result.InputSchema, &schema); err != nil {
		t.Fatalf("TestConvertSchema_QueryAndPathParams: failed to unmarshal schema: %v", err)
	}

	// Verify type is object.
	if got := schema["type"]; got != "object" {
		t.Errorf("TestConvertSchema_QueryAndPathParams: got type: %v, want: object", got)
	}

	// Verify properties exist.
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("TestConvertSchema_QueryAndPathParams: got: missing properties, want: properties map")
	}
	if _, ok := props["id"]; !ok {
		t.Errorf("TestConvertSchema_QueryAndPathParams: got: missing 'id' property, want: 'id' present")
	}
	if _, ok := props["limit"]; !ok {
		t.Errorf("TestConvertSchema_QueryAndPathParams: got: missing 'limit' property, want: 'limit' present")
	}

	// Verify only path param is required.
	reqRaw, ok := schema["required"].([]any)
	if !ok {
		t.Fatalf("TestConvertSchema_QueryAndPathParams: got: missing required array, want: required array")
	}
	gotRequired := make([]string, len(reqRaw))
	for i, v := range reqRaw {
		gotRequired[i] = v.(string)
	}
	wantRequired := []string{"id"}
	if diff := cmp.Diff(wantRequired, gotRequired); diff != "" {
		t.Errorf("TestConvertSchema_QueryAndPathParams: required mismatch (-want, +got): %v", diff)
	}

	// Verify ParamMeta.
	wantMeta := []toolgen.ParamMeta{
		{Name: "id", In: "path", Required: true},
		{Name: "limit", In: "query", Required: false},
	}
	if diff := cmp.Diff(wantMeta, result.ParamMeta); diff != "" {
		t.Errorf("TestConvertSchema_QueryAndPathParams: ParamMeta mismatch (-want, +got): %v", diff)
	}
}

func TestConvertSchema_WithRequestBody(t *testing.T) {
	params := openapi3.Parameters{
		&openapi3.ParameterRef{
			Value: &openapi3.Parameter{
				Name:     "id",
				In:       "path",
				Required: true,
				Schema: &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: &openapi3.Types{"string"},
					},
				},
			},
		},
	}

	body := &openapi3.RequestBodyRef{
		Value: &openapi3.RequestBody{
			Required: true,
			Content: openapi3.Content{
				"application/json": &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type:     &openapi3.Types{"object"},
							Required: []string{"title"},
							Properties: openapi3.Schemas{
								"title": &openapi3.SchemaRef{
									Value: &openapi3.Schema{
										Type: &openapi3.Types{"string"},
									},
								},
								"count": &openapi3.SchemaRef{
									Value: &openapi3.Schema{
										Type: &openapi3.Types{"integer"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	result, err := toolgen.ConvertSchema(params, body)
	if err != nil {
		t.Fatalf("TestConvertSchema_WithRequestBody: got: %v, want: nil error", err)
	}

	var schema map[string]any
	if err := json.Unmarshal(result.InputSchema, &schema); err != nil {
		t.Fatalf("TestConvertSchema_WithRequestBody: failed to unmarshal schema: %v", err)
	}

	// Verify all three properties are present.
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("TestConvertSchema_WithRequestBody: got: missing properties, want: properties map")
	}
	for _, name := range []string{"id", "title", "count"} {
		if _, ok := props[name]; !ok {
			t.Errorf("TestConvertSchema_WithRequestBody: got: missing %q property, want: %q present", name, name)
		}
	}

	// Verify merged required array (sorted).
	reqRaw, ok := schema["required"].([]any)
	if !ok {
		t.Fatalf("TestConvertSchema_WithRequestBody: got: missing required array, want: required array")
	}
	gotRequired := make([]string, len(reqRaw))
	for i, v := range reqRaw {
		gotRequired[i] = v.(string)
	}
	wantRequired := []string{"id", "title"}
	if diff := cmp.Diff(wantRequired, gotRequired); diff != "" {
		t.Errorf("TestConvertSchema_WithRequestBody: required mismatch (-want, +got): %v", diff)
	}

	// Verify ParamMeta includes body params.
	if len(result.ParamMeta) != 3 {
		t.Fatalf("TestConvertSchema_WithRequestBody: got: %d ParamMeta entries, want: 3", len(result.ParamMeta))
	}
	gotPathMeta := result.ParamMeta[0]
	wantPathMeta := toolgen.ParamMeta{Name: "id", In: "path", Required: true}
	if diff := cmp.Diff(wantPathMeta, gotPathMeta); diff != "" {
		t.Errorf("TestConvertSchema_WithRequestBody: path ParamMeta mismatch (-want, +got): %v", diff)
	}

	// Body params should have In="body".
	for _, pm := range result.ParamMeta[1:] {
		if pm.In != "body" {
			t.Errorf("TestConvertSchema_WithRequestBody: got: ParamMeta.In=%q for %q, want: body", pm.In, pm.Name)
		}
	}
}

func TestConvertSchema_CollisionFails(t *testing.T) {
	params := openapi3.Parameters{
		&openapi3.ParameterRef{
			Value: &openapi3.Parameter{
				Name:     "name",
				In:       "path",
				Required: true,
				Schema: &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: &openapi3.Types{"string"},
					},
				},
			},
		},
	}

	body := &openapi3.RequestBodyRef{
		Value: &openapi3.RequestBody{
			Required: true,
			Content: openapi3.Content{
				"application/json": &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type:     &openapi3.Types{"object"},
							Required: []string{"name"},
							Properties: openapi3.Schemas{
								"name": &openapi3.SchemaRef{
									Value: &openapi3.Schema{
										Type: &openapi3.Types{"string"},
									},
								},
								"value": &openapi3.SchemaRef{
									Value: &openapi3.Schema{
										Type: &openapi3.Types{"integer"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := toolgen.ConvertSchema(params, body)
	if err == nil {
		t.Fatalf("TestConvertSchema_CollisionFails: got: nil error, want: collision error")
	}
	if !strings.Contains(err.Error(), "collision") {
		t.Errorf("TestConvertSchema_CollisionFails: got: %v, want: error containing 'collision'", err)
	}
}

func TestConvertSchema_NoParams(t *testing.T) {
	result, err := toolgen.ConvertSchema(nil, nil)
	if err != nil {
		t.Fatalf("TestConvertSchema_NoParams: got: %v, want: nil error", err)
	}

	var schema map[string]any
	if err := json.Unmarshal(result.InputSchema, &schema); err != nil {
		t.Fatalf("TestConvertSchema_NoParams: failed to unmarshal schema: %v", err)
	}

	if got := schema["type"]; got != "object" {
		t.Errorf("TestConvertSchema_NoParams: got type: %v, want: object", got)
	}

	if _, ok := schema["properties"]; ok {
		t.Errorf("TestConvertSchema_NoParams: got: properties present, want: no properties")
	}

	if _, ok := schema["required"]; ok {
		t.Errorf("TestConvertSchema_NoParams: got: required present, want: no required")
	}

	if len(result.ParamMeta) != 0 {
		t.Errorf("TestConvertSchema_NoParams: got: %d ParamMeta entries, want: 0", len(result.ParamMeta))
	}
}
