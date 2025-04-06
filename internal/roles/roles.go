package roles

import (
	"database/sql"
	"fmt"
	"iter"
	"maps"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/pajbot2-discord/internal/serverconfig"
)

type RoleData struct {
	Name        string
	DisplayName string
	Description string
}

var (
	validRoles = map[string]*RoleData{
		"minimod": {
			Name:        "minimod",
			DisplayName: "Mini moderator",
			Description: "",
		},
		"mod": {
			Name:        "mod",
			DisplayName: "Moderator",
			Description: "",
		},
		"admin": {
			Name:        "admin",
			DisplayName: "Admin",
			Description: "",
		},
		"muted": {
			Name:        "muted",
			DisplayName: "Muted",
			Description: "The role to use for the /mute command",
		},
		"nitrobooster": {
			Name:        "nitrobooster",
			DisplayName: "Nitro booster",
			Description: "",
		},
		"member": {
			Name:        "member",
			DisplayName: "Member",
			Description: "",
		},
	}
)

func List() iter.Seq[*RoleData] {
	return maps.Values(validRoles)
}

func Choices() []*discordgo.ApplicationCommandOptionChoice {
	choices := []*discordgo.ApplicationCommandOptionChoice{}

	for roleKey, roleData := range validRoles {
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  roleData.DisplayName,
			Value: roleKey,
		})
	}

	return choices
}

func init() {
	if len(validRoles) > 24 {
		panic("Too many roles, things will break")
	}
}

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

// Grant the given User ID the role
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

func Set(sqlClient *sql.DB, guildID, roleName, roleID string) error {
	if !Valid(roleName) {
		return fmt.Errorf("invalid role name: %s", roleName)
	}

	if roleID == "" {
		return fmt.Errorf("missing role ID")
	}

	key := fmt.Sprintf("role:%s", roleName)

	return serverconfig.Save(sqlClient, guildID, key, roleID)
}

func Clear(sqlClient *sql.DB, guildID, roleName string) error {
	if !Valid(roleName) {
		return fmt.Errorf("invalid role name: %s", roleName)
	}

	key := fmt.Sprintf("role:%s", roleName)

	return serverconfig.Remove(sqlClient, guildID, key)
}
