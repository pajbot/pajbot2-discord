package ping

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
		"$pang",
		"$peng",
		"$ping",
		"$pong",
		"$pung",
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

var vowels = []rune{
	'a', 'e', 'i', 'o', 'u',
}

func (c *Command) Run(s *discordgo.Session, m *discordgo.MessageCreate, parts []string) pkg.CommandResult {
	response := fmt.Sprintf("%s, p%cng", m.Author.Mention(), vowels[rand.Intn(len(vowels))])
	s.ChannelMessageSend(m.ChannelID, response)
	return pkg.CommandResultFullCooldown
}

func (c *Command) Description() string {
	return c.Command.Description
}
