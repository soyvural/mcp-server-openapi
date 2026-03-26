package toolgen

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// LoadSpec loads and validates an OpenAPI spec.
func LoadSpec(ctx context.Context, source string) (*openapi3.T, error) {
	loader := openapi3.NewLoader()
	loader.Context = ctx

	var doc *openapi3.T
	var err error

	if isURL(source) {
		u, parseErr := url.Parse(source)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid URL %q: %w", source, parseErr)
		}
		doc, err = loader.LoadFromURI(u)
	} else {
		doc, err = loader.LoadFromFile(source)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load spec from %q: %w", source, err)
	}

	if err := doc.Validate(ctx); err != nil {
		return nil, fmt.Errorf("spec validation failed: %w", err)
	}

	return doc, nil
}

// ServerURL returns the base API URL from spec or override.
func ServerURL(doc *openapi3.T, override string) (string, error) {
	if override != "" {
		return strings.TrimRight(override, "/"), nil
	}
	if len(doc.Servers) > 0 {
		return strings.TrimRight(doc.Servers[0].URL, "/"), nil
	}
	return "", fmt.Errorf("no server URL in spec and no --server-url provided")
}

func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}
