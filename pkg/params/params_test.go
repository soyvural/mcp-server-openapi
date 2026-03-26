package params

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRequired(t *testing.T) {
	t.Helper()

	tests := []struct {
		desc    string
		args    map[string]any
		key     string
		want    any
		wantErr bool
	}{
		{
			desc: "string key exists",
			args: map[string]any{"name": "test"},
			key:  "name",
			want: "test",
		},
		{
			desc:    "key missing",
			args:    map[string]any{},
			key:     "missing",
			wantErr: true,
		},
		{
			desc:    "wrong type",
			args:    map[string]any{"count": 42},
			key:     "count",
			want:    "",
			wantErr: true,
		},
		{
			desc: "int key exists",
			args: map[string]any{"count": 42},
			key:  "count",
			want: 42,
		},
		{
			desc: "float64 from json",
			args: map[string]any{"value": 3.14},
			key:  "value",
			want: 3.14,
		},
		{
			desc: "bool key exists",
			args: map[string]any{"enabled": true},
			key:  "enabled",
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			t.Helper()

			var got any
			var err error

			switch tc.want.(type) {
			case string:
				got, err = Required[string](tc.args, tc.key)
			case int:
				got, err = Required[int](tc.args, tc.key)
			case float64:
				got, err = Required[float64](tc.args, tc.key)
			case bool:
				got, err = Required[bool](tc.args, tc.key)
			default:
				got, err = Required[string](tc.args, tc.key)
			}

			if tc.wantErr {
				if err == nil {
					t.Errorf("%s: got: nil error, want: error", tc.desc)
				}
				return
			}

			if err != nil {
				t.Errorf("%s: got: error %v, want: nil", tc.desc, err)
				return
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("%s: value mismatch (-want, +got): %v", tc.desc, diff)
			}
		})
	}
}

func TestOptional(t *testing.T) {
	t.Helper()

	tests := []struct {
		desc       string
		args       map[string]any
		key        string
		defaultVal any
		want       any
	}{
		{
			desc:       "string key exists",
			args:       map[string]any{"name": "test"},
			key:        "name",
			defaultVal: "default",
			want:       "test",
		},
		{
			desc:       "key missing returns default",
			args:       map[string]any{},
			key:        "missing",
			defaultVal: "default",
			want:       "default",
		},
		{
			desc:       "wrong type returns default",
			args:       map[string]any{"count": 42},
			key:        "count",
			defaultVal: "default",
			want:       "default",
		},
		{
			desc:       "int key exists",
			args:       map[string]any{"count": 42},
			key:        "count",
			defaultVal: 0,
			want:       42,
		},
		{
			desc:       "int key missing",
			args:       map[string]any{},
			key:        "count",
			defaultVal: 10,
			want:       10,
		},
		{
			desc:       "float64 from json",
			args:       map[string]any{"value": 3.14},
			key:        "value",
			defaultVal: 0.0,
			want:       3.14,
		},
		{
			desc:       "float64 wrong type",
			args:       map[string]any{"value": "not a number"},
			key:        "value",
			defaultVal: 1.0,
			want:       1.0,
		},
		{
			desc:       "bool key exists",
			args:       map[string]any{"enabled": true},
			key:        "enabled",
			defaultVal: false,
			want:       true,
		},
		{
			desc:       "bool key missing",
			args:       map[string]any{},
			key:        "enabled",
			defaultVal: false,
			want:       false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			t.Helper()

			var got any

			switch tc.defaultVal.(type) {
			case string:
				got = Optional[string](tc.args, tc.key, tc.defaultVal.(string))
			case int:
				got = Optional[int](tc.args, tc.key, tc.defaultVal.(int))
			case float64:
				got = Optional[float64](tc.args, tc.key, tc.defaultVal.(float64))
			case bool:
				got = Optional[bool](tc.args, tc.key, tc.defaultVal.(bool))
			default:
				got = Optional[string](tc.args, tc.key, tc.defaultVal.(string))
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("%s: value mismatch (-want, +got): %v", tc.desc, diff)
			}
		})
	}
}
