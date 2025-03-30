package roles

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/pajbot2-discord/internal/serverconfig"
)

var (
	validRoles = map[string]bool{
		"minimod":      true,
		"mod":          true,
		"admin":        true,
		"muted":        true,
		"nitrobooster": true,
		"member":       true,
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

func Grant(s *discordgo.Session, guildID, userID, roleName string) error {
	roleID := GetSingle(guildID, roleName)
	if roleID == "" {
		return fmt.Errorf("no role '%s' set in %s", roleName, guildID)
	}

	fmt.Printf("Granting %s role to %s\n", roleName, userID)
	if err := s.GuildMemberRoleAdd(guildID, userID, roleID); err != nil {
		return fmt.Errorf("role add %s %s: %w", roleName, userID, err)
	}

	return nil
}
