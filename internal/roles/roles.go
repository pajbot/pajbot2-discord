package roles

import "github.com/pajbot/pajbot2-discord/internal/serverconfig"

var (
	validRoles = map[string]bool{
		"minimod":      true,
		"mod":          true,
		"admin":        true,
		"muted":        true,
		"nitrobooster": true,
	}
)

func Valid(role string) (ok bool) {
	_, ok = validRoles[role]
	return
}

func GetSingle(serverID, role string) (roleID string) {
	if !Valid(role) {
		return ""
	}
	key := "role:" + role
	return serverconfig.Get(serverID, key)
}

func Get(serverID, roleName string) (roleIDs []string) {
	roles := resolve(roleName)
	for _, roleName := range roles {
		roleID := GetSingle(serverID, roleName)
		if roleID == "" {
			continue
		}
		roleIDs = append(roleIDs, roleID)
	}

	return
}
