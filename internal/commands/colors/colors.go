package roleinfo

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/basecommand"
	"github.com/pajbot/pajbot2-discord/pkg"
	"github.com/pajbot/pajbot2-discord/pkg/commands"
)

func init() {
	commands.Register([]string{"$colors"}, New())
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

func (c *Command) Run(s *discordgo.Session, m *discordgo.MessageCreate, parts []string) pkg.CommandResult {
	res = pkg.CommandResultUserCooldown
	const usage = `$colors`

	roleNames := strings.Join(config.ColorPickerNames[0:], ", ")

	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("These are available colors: %s", roleNames))
	return pkg.CommandResultUserCooldown
}

func (c *Command) Description() string {
	return c.Command.Description
}
