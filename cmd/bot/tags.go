package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	c2 "github.com/pajlada/pajbot2/pkg/commands"
)

var _ Command = &cmdTags{}

type cmdTags struct {
	c2.Base
}

func newTags() *cmdTags {
	return &cmdTags{
		Base: c2.NewBase(),
	}
}

func (c *cmdTags) Run(s *discordgo.Session, m *discordgo.MessageCreate, parts []string) (res CommandResult) {
	res = CommandResultUserCooldown

	if m.Author != nil {
		const responseFormat = "%s, your user tags are: ID=`%s`, Name=`%s`, Discriminator=`%s`, Verified=`%t`, Bot=`%t`"
		response := fmt.Sprintf(responseFormat, m.Author.Mention(), m.Author.ID, m.Author.Username, m.Author.Discriminator, m.Author.Verified, m.Author.Bot)
		s.ChannelMessageSend(m.ChannelID, response)
	}

	return
}

func (c *cmdTags) Description() string {
	return c.Base.Description
}
