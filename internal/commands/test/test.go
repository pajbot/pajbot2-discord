package mute

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/basecommand"
	"github.com/pajbot/pajbot2-discord/internal/config"
	"github.com/pajbot/pajbot2-discord/internal/mute"
	"github.com/pajbot/pajbot2-discord/pkg"
	"github.com/pajbot/pajbot2-discord/pkg/commands"
	"github.com/pajbot/pajbot2-discord/pkg/utils"
)

func init() {
	commands.Register([]string{"$test"}, New())
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
	hasAccess, err := utils.MemberInRoles(s, m.GuildID, m.Author.ID, config.MiniModeratorRoles)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	if !hasAccess {
		fmt.Println("No access")
		return
	}

	if len(m.Mentions) == 0 {
		return
	}

	target := m.Mentions[0]

	muted, err := mute.IsUserMuted(commands.SQLClient, m.GuildID, target.ID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, m.Author.Mention()+" error checking mute status: "+err.Error())
		return
	}

	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s, user is %v", m.Author.Mention(), muted))

	return
}

func (c *Command) Description() string {
	return c.Command.Description
}
