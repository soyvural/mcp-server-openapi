package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/soyvural/mcp-server-openapi/toolgen"
)

const maxResponseBody = 4096

// ToolRequest holds the data needed to call an upstream API.
type ToolRequest struct {
	ServerURL string
	Path      string
	Method    string
	Args      map[string]any
	ParamMeta []toolgen.ParamMeta
}

// ToolResponse holds the upstream API response.
type ToolResponse struct {
	StatusCode int
	Body       string
	IsError    bool
}

// RequestExecutor makes HTTP calls for tool invocations.
type RequestExecutor interface {
	Execute(ctx context.Context, req *ToolRequest) (*ToolResponse, error)
}

// HTTPExecutor implements RequestExecutor using net/http.
type HTTPExecutor struct {
	client  *http.Client
	auth    Authenticator
	timeout time.Duration
}

// New creates an HTTPExecutor.
func New(client *http.Client, auth Authenticator, timeout time.Duration) *HTTPExecutor {
	if client == nil {
		client = &http.Client{}
	}
	return &HTTPExecutor{client: client, auth: auth, timeout: timeout}
}

// Execute runs the tool request against the upstream API.
func (e *HTTPExecutor) Execute(ctx context.Context, req *ToolRequest) (*ToolResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	fullURL, body, err := buildRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	slog.Debug("executing tool request", "method", req.Method, "url", fullURL)

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	if body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	if err := e.auth.Apply(httpReq); err != nil {
		return nil, fmt.Errorf("failed to apply auth: %w", err)
	}

	resp, err := e.client.Do(httpReq)
	if err != nil {
		if ctx.Err() != nil {
			return &ToolResponse{Body: "request timed out", IsError: true}, nil
		}
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	slog.Debug("tool request completed", "status", resp.StatusCode, "body_len", len(respBody))

	return &ToolResponse{
		StatusCode: resp.StatusCode,
		Body:       string(respBody),
		IsError:    resp.StatusCode >= 400,
	}, nil
}

// buildRequest decomposes tool args into path, query, and body params.
func buildRequest(req *ToolRequest) (string, []byte, error) {
	pathParams := make(map[string]string)
	queryParams := url.Values{}
	bodyParams := make(map[string]any)

	metaByName := make(map[string]string, len(req.ParamMeta))
	for _, pm := range req.ParamMeta {
		metaByName[pm.Name] = pm.In
	}

	for k, v := range req.Args {
		loc, ok := metaByName[k]
		if !ok {
			bodyParams[k] = v
			continue
		}
		switch loc {
		case "path":
			pathParams[k] = fmt.Sprintf("%v", v)
		case "query":
			queryParams.Set(k, fmt.Sprintf("%v", v))
		default:
			bodyParams[k] = v
		}
	}

	path := req.Path
	for k, v := range pathParams {
		path = strings.ReplaceAll(path, "{"+k+"}", url.PathEscape(v))
	}

	fullURL := req.ServerURL + path
	if len(queryParams) > 0 {
		fullURL += "?" + queryParams.Encode()
	}

	var body []byte
	if len(bodyParams) > 0 {
		var err error
		body, err = json.Marshal(bodyParams)
		if err != nil {
			return "", nil, fmt.Errorf("failed to marshal body params: %w", err)
		}
	}

	return fullURL, body, nil
}
