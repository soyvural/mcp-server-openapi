package executor

import (
	"fmt"
	"net/http"
	"os"
)

// Authenticator injects auth into outbound HTTP requests.
type Authenticator interface {
	Apply(req *http.Request) error
}

// NoAuth is a no-op authenticator.
type NoAuth struct{}

func (*NoAuth) Apply(_ *http.Request) error { return nil }

// BearerAuth adds Authorization: Bearer from an env var.
type BearerAuth struct {
	tokenEnv string
}

func NewBearerAuth(tokenEnv string) *BearerAuth {
	return &BearerAuth{tokenEnv: tokenEnv}
}

func (a *BearerAuth) Apply(req *http.Request) error {
	token := os.Getenv(a.tokenEnv)
	if token == "" {
		return fmt.Errorf("bearer token env var %q is empty or not set", a.tokenEnv)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return nil
}

// APIKeyAuth adds an API key as a header or query param from an env var.
type APIKeyAuth struct {
	keyEnv string
	in     string
	name   string
}

func NewAPIKeyAuth(keyEnv, in, name string) *APIKeyAuth {
	return &APIKeyAuth{keyEnv: keyEnv, in: in, name: name}
}

func (a *APIKeyAuth) Apply(req *http.Request) error {
	key := os.Getenv(a.keyEnv)
	if key == "" {
		return fmt.Errorf("API key env var %q is empty or not set", a.keyEnv)
	}
	switch a.in {
	case "header":
		req.Header.Set(a.name, key)
	case "query":
		q := req.URL.Query()
		q.Set(a.name, key)
		req.URL.RawQuery = q.Encode()
	default:
		return fmt.Errorf("unsupported API key location: %q (must be header or query)", a.in)
	}
	return nil
}

// NewAuthenticator creates the right authenticator from CLI flags.
func NewAuthenticator(authType, tokenEnv, keyEnv, keyName, keyIn string) Authenticator {
	switch authType {
	case "bearer":
		return NewBearerAuth(tokenEnv)
	case "api-key":
		return NewAPIKeyAuth(keyEnv, keyIn, keyName)
	default:
		return &NoAuth{}
	}
}
