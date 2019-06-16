package ban

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/basecommand"
	"github.com/pajbot/pajbot2-discord/internal/config"
	"github.com/pajbot/pajbot2-discord/pkg"
	"github.com/pajbot/pajbot2-discord/pkg/commands"
)

func init() {
	commands.Register([]string{"$ban"}, New())
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
	res = pkg.CommandResultNoCooldown
	// FIXME
	// hasAccess, err := memberInRoles(s, m.GuildID, m.Author.ID, moderatorRoles)
	hasAccess := false
	var err error
	if err != nil {
		fmt.Println("Error:", err)
		return pkg.CommandResultUserCooldown
	}

	if !hasAccess {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s, you don't have permission dummy", m.Author.Mention()))
		return pkg.CommandResultUserCooldown
	}

	if len(m.Mentions) == 0 {
		s.ChannelMessageSend(m.ChannelID, "missing user arg. usage: $ban <user> <reason>")
		return
	}

	target := m.Mentions[0]

	if len(parts) < 3 {
		s.ChannelMessageSend(m.ChannelID, "missing reason arg. usage: $ban <user> <reason>")
		return
	}

	reason := strings.Join(parts[2:], " ")

	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Banning %s (%s) for reason: `%s`", target.Username, target.ID, reason))
	s.ChannelMessageSend(config.ModerationActionChannelID, fmt.Sprintf("%s banned %s (%s) for reason: `%s`", m.Author.Username, target.Username, target.ID, reason))
	s.GuildBanCreateWithReason(m.GuildID, target.ID, reason, 0)

	return
}

func (c *Command) Description() string {
	return c.Command.Description
}
