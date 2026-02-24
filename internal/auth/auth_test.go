package auth

import (
	"testing"
)

func TestExtractGroups(t *testing.T) {
	tests := []struct {
		name   string
		claims map[string]interface{}
		want   int
	}{
		{
			name:   "no groups claim",
			claims: map[string]interface{}{"sub": "user1"},
			want:   0,
		},
		{
			name: "groups as []interface{}",
			claims: map[string]interface{}{
				"groups": []interface{}{"admin", "users"},
			},
			want: 2,
		},
		{
			name: "groups as []string",
			claims: map[string]interface{}{
				"groups": []string{"admin"},
			},
			want: 1,
		},
		{
			name: "groups with non-string values",
			claims: map[string]interface{}{
				"groups": []interface{}{"admin", 42, "users"},
			},
			want: 2,
		},
		{
			name: "groups as wrong type",
			claims: map[string]interface{}{
				"groups": "single-string",
			},
			want: 0,
		},
		{
			name:   "nil claims",
			claims: nil,
			want:   0,
		},
		{
			name: "empty groups",
			claims: map[string]interface{}{
				"groups": []interface{}{},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractGroups(tt.claims)
			if len(got) != tt.want {
				t.Errorf("ExtractGroups() returned %d groups, want %d", len(got), tt.want)
			}
		})
	}
}

func TestMapGroupsToRole(t *testing.T) {
	tests := []struct {
		name   string
		groups []string
		cfg    OIDCGroupConfig
		want   string
	}{
		{
			name:   "admin group match",
			groups: []string{"bookowl-admins", "bookowl-users"},
			cfg: OIDCGroupConfig{
				AdminGroups:  []string{"bookowl-admins"},
				EditorGroups: []string{"bookowl-editors"},
			},
			want: "admin",
		},
		{
			name:   "editor group match",
			groups: []string{"bookowl-editors", "bookowl-users"},
			cfg: OIDCGroupConfig{
				AdminGroups:  []string{"bookowl-admins"},
				EditorGroups: []string{"bookowl-editors"},
			},
			want: "editor",
		},
		{
			name:   "admin takes precedence over editor",
			groups: []string{"bookowl-admins", "bookowl-editors"},
			cfg: OIDCGroupConfig{
				AdminGroups:  []string{"bookowl-admins"},
				EditorGroups: []string{"bookowl-editors"},
			},
			want: "admin",
		},
		{
			name:   "no group config → viewer",
			groups: []string{"random-group"},
			cfg:    OIDCGroupConfig{},
			want:   "viewer",
		},
		{
			name:   "allowed groups — in list → viewer",
			groups: []string{"bookowl-users"},
			cfg: OIDCGroupConfig{
				AllowedGroups: []string{"bookowl-users"},
			},
			want: "viewer",
		},
		{
			name:   "allowed groups — not in list → deny",
			groups: []string{"other-group"},
			cfg: OIDCGroupConfig{
				AllowedGroups: []string{"bookowl-users"},
			},
			want: "",
		},
		{
			name:   "empty groups — no restrictions → viewer",
			groups: []string{},
			cfg:    OIDCGroupConfig{},
			want:   "viewer",
		},
		{
			name:   "empty groups — with allowed groups → deny",
			groups: []string{},
			cfg: OIDCGroupConfig{
				AllowedGroups: []string{"bookowl-users"},
			},
			want: "",
		},
		{
			name:   "multiple admin groups",
			groups: []string{"platform-leads"},
			cfg: OIDCGroupConfig{
				AdminGroups: []string{"bookowl-admins", "platform-leads"},
			},
			want: "admin",
		},
		{
			name:   "allowed + editor → editor wins",
			groups: []string{"bookowl-editors", "bookowl-users"},
			cfg: OIDCGroupConfig{
				EditorGroups:  []string{"bookowl-editors"},
				AllowedGroups: []string{"bookowl-users", "bookowl-editors"},
			},
			want: "editor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapGroupsToRole(tt.groups, tt.cfg)
			if got != tt.want {
				t.Errorf("MapGroupsToRole() = %q, want %q", got, tt.want)
			}
		})
	}
}
