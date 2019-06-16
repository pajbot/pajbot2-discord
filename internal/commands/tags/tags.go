package tags

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/basecommand"
	"github.com/pajbot/pajbot2-discord/pkg"
	"github.com/pajbot/pajbot2-discord/pkg/commands"
)

func init() {
	commands.Register([]string{"$tags"}, New())
}

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

	if m.Author != nil {
		const responseFormat = "%s, your user tags are: ID=`%s`, Name=`%s`, Discriminator=`%s`, Verified=`%t`, Bot=`%t`"
		response := fmt.Sprintf(responseFormat, m.Author.Mention(), m.Author.ID, m.Author.Username, m.Author.Discriminator, m.Author.Verified, m.Author.Bot)
		s.ChannelMessageSend(m.ChannelID, response)
	}

	return
}

func (c *Command) Description() string {
	return c.Command.Description
}
