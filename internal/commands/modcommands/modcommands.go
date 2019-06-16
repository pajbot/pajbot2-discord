package modcommands

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
	commands.Register([]string{"$modcommands"}, New())
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
	hasAccess, err := utils.MemberInRoles(s, m.GuildID, m.Author.ID, config.MiniModeratorRoles)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	if !hasAccess {
		// No permission
		return
	}

	// FIXME: implement
	// descriptions := []string{}
	// commands.ForEach(func(aliases []string, iCmd interface{}) {
	// 	var description string
	// 	if cmd, ok := iCmd.(Command); ok {
	// 		description = fmt.Sprintf("`%s`: %s", aliases, cmd.Description())
	// 	}

	// 	if description != "" {
	// 		descriptions = append(descriptions, description)
	// 	}
	// })

	// utils.SendChunks("", "", descriptions, m.ChannelID, s)
	return
}

func (c *Command) Description() string {
	return c.Base.Description
}
