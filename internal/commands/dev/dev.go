package dev

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/basecommand"
	"github.com/pajbot/pajbot2-discord/pkg"
	"github.com/pajbot/pajbot2-discord/pkg/commands"
	"github.com/pajbot/pajbot2-discord/pkg/utils"
)

var _ pkg.Command = &Command{}

func init() {
	commands.Register([]string{
		"$dev",
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

func (c *Command) Run(s *discordgo.Session, m *discordgo.MessageCreate, parts []string) (res pkg.CommandResult) {
	res = pkg.CommandResultNoCooldown
	hasAccess, err := utils.MemberHasPermission(s, m.GuildID, m.Author.ID, "mod")
	if err != nil {
		fmt.Println("Error:", err)
		return pkg.CommandResultUserCooldown
	}

	if !hasAccess {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s, you don't have permission dummy", m.Author.Mention()))
		return pkg.CommandResultUserCooldown
	}

	const a = "a `b` c"
	const b = "a _b_ c"

	tests := []func(){
		func() {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("hi _hi_ %s", utils.EscapeMarkdown("_hi_")))
		},
		func() {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf(
				"2: %s `%s`",
				a,
				utils.EscapeMarkdown(a),
			))
		},
		func() {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf(
				"3 raw: %s _%s_ `%s`",
				a,
				a,
				a,
			))
			e := fmt.Sprintf(
				"3 esc: %s _%s_ `%s`",
				utils.EscapeMarkdown(a),
				utils.EscapeMarkdown(a),
				utils.EscapeMarkdown(a),
			)
			s.ChannelMessageSend(m.ChannelID, e)
		},
		func() {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf(
				"4 raw: %s _%s_ `%s`",
				b,
				b,
				b,
			))
			e := fmt.Sprintf(
				"4 esc: %s _%s_ `%s`",
				utils.EscapeMarkdown(b),
				utils.EscapeMarkdown(b),
				utils.EscapeMarkdown(b),
			)
			s.ChannelMessageSend(m.ChannelID, e)
		},
	}

	for i, test := range tests {
		time.AfterFunc(time.Duration(i*500)*time.Millisecond, test)
	}

	return
}

func (c *Command) Description() string {
	return c.Command.Description
}
