package executor

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestBearerAuth(t *testing.T) {
	tests := []struct {
		desc      string
		tokenEnv  string
		tokenVal  string
		wantError bool
		wantAuth  string
	}{
		{
			desc:      "valid bearer token",
			tokenEnv:  "TEST_BEARER_TOKEN",
			tokenVal:  "secret123",
			wantError: false,
			wantAuth:  "Bearer secret123",
		},
		{
			desc:      "empty env var value",
			tokenEnv:  "EMPTY_TOKEN",
			tokenVal:  "",
			wantError: true,
			wantAuth:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.tokenVal != "" {
				t.Setenv(tc.tokenEnv, tc.tokenVal)
			}

			auth := NewBearerAuth(tc.tokenEnv)
			req, err := http.NewRequest("GET", "http://example.com", nil)
			if err != nil {
				t.Fatalf("%s: failed to create request: %v", tc.desc, err)
			}

			err = auth.Apply(req)
			if tc.wantError {
				if err == nil {
					t.Errorf("%s: got: nil error, want: error", tc.desc)
				}
				return
			}

			if err != nil {
				t.Errorf("%s: got: error %v, want: nil", tc.desc, err)
				return
			}

			got := req.Header.Get("Authorization")
			if diff := cmp.Diff(tc.wantAuth, got); diff != "" {
				t.Errorf("%s: Authorization header mismatch (-want, +got):\n%s", tc.desc, diff)
			}
		})
	}
}

func TestBearerAuth_EmptyEnv(t *testing.T) {
	auth := NewBearerAuth("NONEXISTENT_TOKEN")
	req, err := http.NewRequest("GET", "http://example.com", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	err = auth.Apply(req)
	if err == nil {
		t.Errorf("got: nil error, want: error for empty env var")
	}
}

func TestAPIKeyAuth_Header(t *testing.T) {
	tests := []struct {
		desc       string
		keyEnv     string
		keyVal     string
		keyName    string
		wantError  bool
		wantHeader string
	}{
		{
			desc:       "valid API key in header",
			keyEnv:     "TEST_API_KEY",
			keyVal:     "apikey123",
			keyName:    "X-API-Key",
			wantError:  false,
			wantHeader: "apikey123",
		},
		{
			desc:       "custom header name",
			keyEnv:     "CUSTOM_KEY",
			keyVal:     "custom456",
			keyName:    "X-Custom-Auth",
			wantError:  false,
			wantHeader: "custom456",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			t.Setenv(tc.keyEnv, tc.keyVal)

			auth := NewAPIKeyAuth(tc.keyEnv, "header", tc.keyName)
			req, err := http.NewRequest("GET", "http://example.com", nil)
			if err != nil {
				t.Fatalf("%s: failed to create request: %v", tc.desc, err)
			}

			err = auth.Apply(req)
			if tc.wantError {
				if err == nil {
					t.Errorf("%s: got: nil error, want: error", tc.desc)
				}
				return
			}

			if err != nil {
				t.Errorf("%s: got: error %v, want: nil", tc.desc, err)
				return
			}

			got := req.Header.Get(tc.keyName)
			if diff := cmp.Diff(tc.wantHeader, got); diff != "" {
				t.Errorf("%s: header %s mismatch (-want, +got):\n%s", tc.desc, tc.keyName, diff)
			}
		})
	}
}

func TestAPIKeyAuth_Query(t *testing.T) {
	tests := []struct {
		desc      string
		keyEnv    string
		keyVal    string
		keyName   string
		wantError bool
		wantParam string
	}{
		{
			desc:      "valid API key in query",
			keyEnv:    "TEST_QUERY_KEY",
			keyVal:    "querykey789",
			keyName:   "api_key",
			wantError: false,
			wantParam: "querykey789",
		},
		{
			desc:      "custom query param name",
			keyEnv:    "CUSTOM_QUERY_KEY",
			keyVal:    "custom999",
			keyName:   "apiKey",
			wantError: false,
			wantParam: "custom999",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			t.Setenv(tc.keyEnv, tc.keyVal)

			auth := NewAPIKeyAuth(tc.keyEnv, "query", tc.keyName)
			req, err := http.NewRequest("GET", "http://example.com", nil)
			if err != nil {
				t.Fatalf("%s: failed to create request: %v", tc.desc, err)
			}

			err = auth.Apply(req)
			if tc.wantError {
				if err == nil {
					t.Errorf("%s: got: nil error, want: error", tc.desc)
				}
				return
			}

			if err != nil {
				t.Errorf("%s: got: error %v, want: nil", tc.desc, err)
				return
			}

			got := req.URL.Query().Get(tc.keyName)
			if diff := cmp.Diff(tc.wantParam, got); diff != "" {
				t.Errorf("%s: query param %s mismatch (-want, +got):\n%s", tc.desc, tc.keyName, diff)
			}
		})
	}
}

func TestAPIKeyAuth_EmptyEnv(t *testing.T) {
	auth := NewAPIKeyAuth("NONEXISTENT_KEY", "header", "X-API-Key")
	req, err := http.NewRequest("GET", "http://example.com", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	err = auth.Apply(req)
	if err == nil {
		t.Errorf("got: nil error, want: error for empty env var")
	}
}

func TestAPIKeyAuth_UnsupportedLocation(t *testing.T) {
	t.Setenv("TEST_KEY", "value123")

	auth := NewAPIKeyAuth("TEST_KEY", "cookie", "api_key")
	req, err := http.NewRequest("GET", "http://example.com", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	err = auth.Apply(req)
	if err == nil {
		t.Errorf("got: nil error, want: error for unsupported location")
	}
}

func TestNoAuth(t *testing.T) {
	auth := &NoAuth{}
	req, err := http.NewRequest("GET", "http://example.com", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	// Add some headers to ensure NoAuth doesn't modify them.
	req.Header.Set("X-Custom", "value")
	originalURL := req.URL.String()

	err = auth.Apply(req)
	if err != nil {
		t.Errorf("got: error %v, want: nil", err)
	}

	// Verify no Authorization header was added.
	if got := req.Header.Get("Authorization"); got != "" {
		t.Errorf("got: Authorization header %q, want: empty", got)
	}

	// Verify existing headers unchanged.
	if got := req.Header.Get("X-Custom"); got != "value" {
		t.Errorf("got: X-Custom header %q, want: %q", got, "value")
	}

	// Verify URL unchanged.
	if got := req.URL.String(); got != originalURL {
		t.Errorf("got: URL %q, want: %q", got, originalURL)
	}
}

func TestNewAuthenticator(t *testing.T) {
	tests := []struct {
		desc     string
		authType string
		tokenEnv string
		keyEnv   string
		keyName  string
		keyIn    string
		wantType string
	}{
		{
			desc:     "bearer auth",
			authType: "bearer",
			tokenEnv: "TOKEN",
			wantType: "*executor.BearerAuth",
		},
		{
			desc:     "api-key auth",
			authType: "api-key",
			keyEnv:   "KEY",
			keyName:  "X-API-Key",
			keyIn:    "header",
			wantType: "*executor.APIKeyAuth",
		},
		{
			desc:     "no auth (empty type)",
			authType: "",
			wantType: "executor.NoAuth",
		},
		{
			desc:     "no auth (unknown type)",
			authType: "oauth",
			wantType: "executor.NoAuth",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			auth := NewAuthenticator(tc.authType, tc.tokenEnv, tc.keyEnv, tc.keyName, tc.keyIn)

			gotType := ""
			switch auth.(type) {
			case *BearerAuth:
				gotType = "*executor.BearerAuth"
			case *APIKeyAuth:
				gotType = "*executor.APIKeyAuth"
			case *NoAuth:
				gotType = "executor.NoAuth"
			default:
				gotType = "unknown"
			}

			if diff := cmp.Diff(tc.wantType, gotType); diff != "" {
				t.Errorf("%s: authenticator type mismatch (-want, +got):\n%s", tc.desc, diff)
			}
		})
	}
}

func TestAPIKeyAuth_QueryPreservesExisting(t *testing.T) {
	t.Setenv("TEST_KEY", "newkey")

	auth := NewAPIKeyAuth("TEST_KEY", "query", "api_key")

	u, err := url.Parse("http://example.com?existing=value")
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}

	req := &http.Request{
		Method: "GET",
		URL:    u,
		Header: make(http.Header),
	}

	err = auth.Apply(req)
	if err != nil {
		t.Fatalf("got: error %v, want: nil", err)
	}

	// Verify both existing and new param present.
	if got := req.URL.Query().Get("existing"); got != "value" {
		t.Errorf("got: existing param %q, want: %q", got, "value")
	}
	if got := req.URL.Query().Get("api_key"); got != "newkey" {
		t.Errorf("got: api_key param %q, want: %q", got, "newkey")
	}
}
