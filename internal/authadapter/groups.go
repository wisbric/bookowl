package authadapter

// ExtractGroups extracts the groups claim from JWT claims.
func ExtractGroups(claims map[string]interface{}) []string {
	raw, ok := claims["groups"]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []interface{}:
		groups := make([]string, 0, len(v))
		for _, g := range v {
			if s, ok := g.(string); ok {
				groups = append(groups, s)
			}
		}
		return groups
	case []string:
		return v
	}
	return nil
}

// OIDCGroupConfig holds the group mapping config from tenant OIDC settings.
type OIDCGroupConfig struct {
	AllowedGroups []string
	AdminGroups   []string
	EditorGroups  []string
}

// MapGroupsToRole maps JWT group membership to a BookOwl role.
// Returns empty string if user should be denied access.
func MapGroupsToRole(groups []string, cfg OIDCGroupConfig) string {
	for _, g := range groups {
		for _, adminGroup := range cfg.AdminGroups {
			if g == adminGroup {
				return "admin"
			}
		}
	}
	for _, g := range groups {
		for _, editorGroup := range cfg.EditorGroups {
			if g == editorGroup {
				return "editor"
			}
		}
	}
	if len(cfg.AllowedGroups) > 0 {
		for _, g := range groups {
			for _, allowed := range cfg.AllowedGroups {
				if g == allowed {
					return "viewer"
				}
			}
		}
		return "" // not in any allowed group → deny
	}
	return "viewer" // no group restrictions → everyone is viewer
}
