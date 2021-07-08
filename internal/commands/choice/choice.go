package choice

import (
	"fmt"
	"math/rand"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/basecommand"
	"github.com/pajbot/pajbot2-discord/pkg"
	"github.com/pajbot/pajbot2-discord/pkg/commands"
)

var _ pkg.Command = &Command{}

func init() {
	commands.Register([]string{
		"$choice",
	}, New())
}

type Command struct {
	basecommand.Command
}

func New() *Command {
	return &Command{
		Command: basecommand.New(),
	}
}

func (c *Command) Run(s *discordgo.Session, m *discordgo.MessageCreate, parts []string) pkg.CommandResult {
	if len(parts) > 2 {
		parts = parts[1:]
		response := fmt.Sprintf("%s, %s", m.Author.Mention(), parts[rand.Intn(len(parts))])
		s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend {
				Content:         response,
				AllowedMentions: &discordgo.MessageAllowedMentions{Users: nil},
		})
	}
	return pkg.CommandResultFullCooldown
}

func (c *Command) Description() string {
	return c.Command.Description
}
