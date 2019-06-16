package roleinfo

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/basecommand"
	"github.com/pajlada/pajbot2-discord/pkg"
	"github.com/pajlada/pajbot2-discord/pkg/commands"
)

func init() {
	commands.Register([]string{"$roleinfo"}, New())
}

var _ pkg.Command = &Command{}

type Command struct {
	basecommand.Command
}

func New() *Command {
	return &Command{
		Command: basecommand.New(),
	}
}

func (c *Command) Run(s *discordgo.Session, m *discordgo.MessageCreate, parts []string) pkg.CommandResult {
	const usage = `$roleinfo ROLENAME (e.g. $roleinfo roleplayer)`

	parts = parts[1:]

	if len(parts) < 1 {
		s.ChannelMessageSend(m.ChannelID, m.Author.Mention()+" usage: "+usage)
		return pkg.CommandResultUserCooldown
	}

	roleName := strings.Join(parts[0:], " ")

	roles, err := s.GuildRoles(m.GuildID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, m.Author.Mention()+" error getting roles: "+err.Error())
		return pkg.CommandResultUserCooldown
	}

	for _, role := range roles {
		if role.Managed {
			continue
		}

		if strings.EqualFold(role.Name, roleName) {
			roleInfoString := fmt.Sprintf("id=%s, color=#%06x", role.ID, role.Color)
			s.ChannelMessageSend(m.ChannelID, m.Author.Mention()+" role info: "+roleInfoString)
			return pkg.CommandResultFullCooldown
		}
	}

	s.ChannelMessageSend(m.ChannelID, m.Author.Mention()+" no role found with that name")
	return pkg.CommandResultUserCooldown
}

func (c *Command) Description() string {
	return c.Command.Description
}
