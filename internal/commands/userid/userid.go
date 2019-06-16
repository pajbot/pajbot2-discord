package userid

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
	commands.Register([]string{"userid"}, New())
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
	// FIXME
	hasAccess, err := utils.MemberInRoles(s, m.GuildID, m.Author.ID, config.MiniModeratorRoles)
	if err != nil {
		fmt.Println("Error:", err)
		return pkg.CommandResultUserCooldown
	}
	if !hasAccess {
		// No permission
		return pkg.CommandResultUserCooldown
	}

	if len(parts) < 2 {
		return
	}

	target := parts[1]
	// FIXME
	targetID := utils.CleanUserID(parts[1])

	s.ChannelMessageSend(m.ChannelID, "User ID of "+target+" is `"+targetID+"`")

	return
}

func (c *Command) Description() string {
	return c.Base.Description
}
