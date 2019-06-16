package ping

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/pajlada/pajbot2-discord/pkg"
	"github.com/pajlada/pajbot2-discord/pkg/commands"
	c2 "github.com/pajlada/pajbot2/pkg/commands"
)

var _ pkg.Command = &Command{}

func init() {
	commands.Register([]string{"$ping"}, New())
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
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s, pong", m.Author.Mention()))
	return pkg.CommandResultFullCooldown
}

func (c *Command) Description() string {
	return c.Base.Description
}
