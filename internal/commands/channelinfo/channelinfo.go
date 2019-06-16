package channelinfo

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/basecommand"
	"github.com/pajlada/pajbot2-discord/pkg"
	"github.com/pajlada/pajbot2-discord/pkg/commands"
)

func init() {
	commands.Register([]string{"$channelinfo"}, New())
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
	msg := fmt.Sprintf("Channel ID: %s", m.ChannelID)
	s.ChannelMessageSend(m.ChannelID, msg)
	return pkg.CommandResultFullCooldown
}

func (c *Command) Description() string {
	return c.Command.Description
}
