package params

import "fmt"

// Required extracts a parameter with type checking, erroring if missing or wrong type.
func Required[T any](args map[string]any, key string) (T, error) {
	var zero T
	v, ok := args[key]
	if !ok {
		return zero, fmt.Errorf("missing required parameter: %s", key)
	}
	typed, ok := v.(T)
	if !ok {
		return zero, fmt.Errorf("parameter %s: expected %T, got %T", key, zero, v)
	}
	return typed, nil
}

// Optional extracts a parameter, returning defaultVal if missing or wrong type.
func Optional[T any](args map[string]any, key string, defaultVal T) T {
	v, ok := args[key]
	if !ok {
		return defaultVal
	}
	typed, ok := v.(T)
	if !ok {
		return defaultVal
	}
	return typed
}
