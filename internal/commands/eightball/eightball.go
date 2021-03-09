package eightball

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
		"$8ball",
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

var replies = []string{
	"As I see it, yes.",
	"Ask again later.",
	"Better not tell you now.",
	"Cannot predict now.",
	"Concentrate and ask again.",
	"Don’t count on it.",
	"It is certain.",
	"It is decidedly so.",
	"Most likely.",
	"My reply is no.",
	"My sources say no.",
	"Outlook not so good.",
	"Outlook good.",
	"Reply hazy, try again.",
	"Signs point to yes.",
	"Very doubtful.",
	"Without a doubt.",
	"Yes.",
	"Yes – definitely.",
	"You may rely on it."
}

func (c *Command) Run(s *discordgo.Session, m *discordgo.MessageCreate, parts []string) pkg.CommandResult {
	response := fmt.Sprintf("%s, %s", m.Author.Mention(), replies[rand.Intn(len(replies))])
	s.ChannelMessageSend(m.ChannelID, response)
	return pkg.CommandResultFullCooldown
}

func (c *Command) Description() string {
	return c.Command.Description
}
