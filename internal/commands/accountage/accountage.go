package accountage

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/basecommand"
	"github.com/pajbot/pajbot2-discord/pkg"
	"github.com/pajbot/pajbot2-discord/pkg/commands"
	"github.com/pajbot/pajbot2-discord/pkg/utils"
)

func init() {
	commands.Register([]string{"$accountage"}, New())
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

func (c *Command) Run(s *discordgo.Session, m *discordgo.MessageCreate, parts []string) (res pkg.CommandResult) {
	res = pkg.CommandResultUserCooldown

	creationTime, err := utils.CreationTime(m.Author.ID)
	if err != nil {
		return
	}

	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s, your account was created at %s", m.Author.Mention(), creationTime))

	return
}

func (c *Command) Description() string {
	return c.Command.Description
}
