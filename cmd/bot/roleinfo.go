package main

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	c2 "github.com/pajlada/pajbot2/pkg/commands"
)

var _ Command = &cmdRoleInfo{}

type cmdRoleInfo struct {
	c2.Base
}

func newRoleInfo() *cmdRoleInfo {
	return &cmdRoleInfo{
		Base: c2.NewBase(),
	}
}

func (c *cmdRoleInfo) Run(s *discordgo.Session, m *discordgo.MessageCreate, parts []string) CommandResult {
	const usage = `$roleinfo ROLENAME (e.g. $roleinfo roleplayer)`

	parts = parts[1:]

	if len(parts) < 1 {
		s.ChannelMessageSend(m.ChannelID, m.Author.Mention()+" usage: "+usage)
		return CommandResultUserCooldown
	}

	roleName := parts[0]

	roles, err := s.GuildRoles(m.GuildID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, m.Author.Mention()+" error getting roles: "+err.Error())
		return CommandResultUserCooldown
	}

	for _, role := range roles {
		if role.Managed {
			continue
		}

		if strings.EqualFold(role.Name, roleName) {
			roleInfoString := fmt.Sprintf("id=%s, color=#%06x", role.ID, role.Color)
			s.ChannelMessageSend(m.ChannelID, m.Author.Mention()+" role info: "+roleInfoString)
			return CommandResultFullCooldown
		}
	}

	s.ChannelMessageSend(m.ChannelID, m.Author.Mention()+" no role found with that name")
	return CommandResultUserCooldown
}

func (c *cmdRoleInfo) Description() string {
	return c.Base.Description
}
