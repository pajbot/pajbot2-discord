package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/pajlada/pajbot2-discord/internal/config"
	"github.com/pajlada/pajbot2-discord/pkg"
	"github.com/pajlada/pajbot2-discord/pkg/commands"
	"github.com/pajlada/pajbot2-discord/pkg/utils"
	c2 "github.com/pajlada/pajbot2/pkg/commands"
)

func init() {
	commands.Register([]string{"$mute"}, New())
}

var _ pkg.Command = &Command{}

type Command struct {
	c2.Base
}

func New() *Command {
	return &Command{
		Base: c2.NewBase(),
	}
}

func (c *Command) Run(s *discordgo.Session, m *discordgo.MessageCreate, parts []string) (res pkg.CommandResult) {
	res = pkg.CommandResultNoCooldown

	hasAccess, err := utils.MemberInRoles(s, m.GuildID, m.Author.ID, config.MiniModeratorRoles)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	if !hasAccess {
		// No permission
		return
	}

	channels, err := s.GuildChannels(m.GuildID)
	if err != nil {
		fmt.Println("Error getting channels:", err)
		return
	}

	outputs := []string{}
	for _, channel := range channels {
		outputs = append(outputs, fmt.Sprintf("[%s] %s = %s\n", utils.GetChannelTypeName(channel.Type), channel.ID, channel.Name))
	}

	utils.SendChunks("```", "```", outputs, m.ChannelID, s)

	return
}

func (c *Command) Description() string {
	return c.Base.Description
}
