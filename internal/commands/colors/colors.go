package roleinfo

import (
	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/basecommand"
	"github.com/pajbot/pajbot2-discord/pkg"
	"github.com/pajbot/pajbot2-discord/pkg/commands"
	"github.com/pajbot/pajbot2-discord/pkg/utils"
	"strings"
)

func init() {
	commands.Register([]string{"$colors", "$colours"}, New())
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
	res = pkg.CommandResultUserCooldown
	const usage = `$colors`

	colorRoles := utils.ColorRoles(s, m.GuildID)
	var roleNames []string
	for _, role := range colorRoles {
		roleNames = append(roleNames, role.Mention())
	}

	if len(roleNames) == 0 {
		s.ChannelMessageSend(m.ChannelID, "No nitro colors set up")
		return
	}

	s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
		Title:       "These are the available colors:",
		Description: strings.Join(roleNames, " "),
	})
	return pkg.CommandResultUserCooldown
}

func (c *Command) Description() string {
	return c.Command.Description
}
