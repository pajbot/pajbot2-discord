package roles

var (
	roleGroups = map[string][]string{
		"minimod": {"minimod", "mod", "admin"},
		"mod":     {"mod", "admin"},
	}
)

func resolve(roleName string) (roles []string) {
	if roles, ok := roleGroups[roleName]; ok {
		return roles
	}

	return []string{roleName}
}
