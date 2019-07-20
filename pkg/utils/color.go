package utils

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func FilterColorRoles(roles []*discordgo.Role) (colorRoles []*discordgo.Role) {
	for _, role := range roles {
		if strings.HasPrefix(role.Name, "nitro:") {
			colorRoles = append(colorRoles, role)
		}
	}

	return
}

func ColorRoles(s *discordgo.Session, guildID string) (colorRoles []*discordgo.Role) {
	roles, err := s.GuildRoles(guildID)
	if err != nil {
		fmt.Println("Error getting roles:", err)
		return
	}

	return FilterColorRoles(roles)
}

func RemoveNitroColors(s *discordgo.Session, guildID, userID string, colorRoles []*discordgo.Role) error {
	member, err := s.GuildMember(guildID, userID)
	if err != nil {
		return err
	}

	// Remove all nitro roles from user
	for _, role := range colorRoles {
		for _, memberRoleID := range member.Roles {
			if role.ID == memberRoleID {
				err = s.GuildMemberRoleRemove(guildID, userID, role.ID)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
