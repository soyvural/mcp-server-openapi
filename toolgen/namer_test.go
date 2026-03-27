package toolgen_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/soyvural/mcp-server-openapi/toolgen"
)

func TestGenerateToolName(t *testing.T) {
	tests := []struct {
		desc        string
		path        string
		method      string
		operationID string
		xMCPName    string
		want        string
	}{
		{
			desc:     "x-mcp-tool-name takes priority",
			path:     "/users/{id}",
			method:   "GET",
			xMCPName: "custom_get_user",
			want:     "custom_get_user",
		},
		{
			desc:        "operationId fallback when x-mcp-tool-name empty",
			path:        "/users/{id}",
			method:      "GET",
			operationID: "getUserById",
			xMCPName:    "",
			want:        "getuserbyid",
		},
		{
			desc:        "method_path fallback when both empty",
			path:        "/users/{id}",
			method:      "GET",
			operationID: "",
			xMCPName:    "",
			want:        "get_users_id",
		},
		{
			desc:        "special characters sanitized",
			path:        "/api/v2/resource-items",
			method:      "POST",
			operationID: "",
			xMCPName:    "",
			want:        "post_api_v2_resource_items",
		},
		{
			desc:     "slashes and braces handled",
			path:     "/api/{version}/users/{userId}/posts/{postId}",
			method:   "DELETE",
			xMCPName: "",
			want:     "delete_api_version_users_userid_posts_postid",
		},
		{
			desc:     "uppercase in x-mcp-tool-name lowercased",
			xMCPName: "CustomToolName",
			want:     "customtoolname",
		},
		{
			desc:        "hyphens in operationId converted to underscores",
			operationID: "list-all-users",
			want:        "list_all_users",
		},
		{
			desc:     "dots preserved",
			xMCPName: "tool.name.with.dots",
			want:     "tool.name.with.dots",
		},
		{
			desc:        "consecutive underscores collapsed",
			operationID: "get__user__by__id",
			want:        "get_user_by_id",
		},
		{
			desc:        "leading and trailing underscores removed",
			operationID: "__getUserById__",
			want:        "getuserbyid",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			got := toolgen.GenerateToolName(tc.path, tc.method, tc.operationID, tc.xMCPName)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("%s: mismatch (-want, +got):\n%s", tc.desc, diff)
			}
		})
	}
}

func TestSanitizeToolName(t *testing.T) {
	tests := []struct {
		desc  string
		input string
		want  string
	}{
		{
			desc:  "already valid",
			input: "get_user_by_id",
			want:  "get_user_by_id",
		},
		{
			desc:  "uppercase converted to lowercase",
			input: "GetUserById",
			want:  "getuserbyid",
		},
		{
			desc:  "hyphens converted to underscores",
			input: "list-all-users",
			want:  "list_all_users",
		},
		{
			desc:  "consecutive underscores collapsed",
			input: "get___user___by___id",
			want:  "get_user_by_id",
		},
		{
			desc:  "leading and trailing underscores removed",
			input: "___get_user___",
			want:  "get_user",
		},
		{
			desc:  "dots preserved",
			input: "tool.name.with.dots",
			want:  "tool.name.with.dots",
		},
		{
			desc:  "slashes converted to underscores",
			input: "api/v1/users",
			want:  "api_v1_users",
		},
		{
			desc:  "braces removed",
			input: "users/{id}/posts/{postId}",
			want:  "users_id_posts_postid",
		},
		{
			desc:  "special characters replaced with underscores",
			input: "get@user#by$id%",
			want:  "get_user_by_id",
		},
		{
			desc:  "mixed special characters and spaces",
			input: "My Tool-Name (v2.0)!",
			want:  "my_tool_name_v2.0",
		},
		{
			desc:  "empty string",
			input: "",
			want:  "",
		},
		{
			desc:  "only special characters",
			input: "!!!@@@###",
			want:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			got := toolgen.SanitizeToolName(tc.input)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("%s: mismatch (-want, +got):\n%s", tc.desc, diff)
			}
		})
	}
}
