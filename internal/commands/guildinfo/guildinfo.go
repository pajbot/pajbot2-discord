package guildinfo

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/pajlada/pajbot2-discord/pkg"
	"github.com/pajlada/pajbot2-discord/pkg/commands"
	c2 "github.com/pajlada/pajbot2/pkg/commands"
)

var _ pkg.Command = &Command{}

func init() {
	commands.Register([]string{"$guildinfo", "$serverinfo"}, New())
}

type Command struct {
	c2.Base
}

func New() *Command {
	return &Command{
		Base: c2.NewBase(),
	}
}

func (c *Command) Run(s *discordgo.Session, m *discordgo.MessageCreate, parts []string) pkg.CommandResult {
	msg := fmt.Sprintf("Server ID: %s", m.GuildID)
	s.ChannelMessageSend(m.ChannelID, msg)
	return pkg.CommandResultFullCooldown
}

func (c *Command) Description() string {
	return c.Base.Description
}
