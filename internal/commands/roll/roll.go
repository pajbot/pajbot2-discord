package roll

import (
	"fmt"
	"math/rand"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/basecommand"
	"github.com/pajbot/pajbot2-discord/pkg"
	"github.com/pajbot/pajbot2-discord/pkg/commands"
	"github.com/pajbot/pajbot2-discord/pkg/utils"
)

var _ pkg.Command = &Command{}

func init() {
	commands.Register([]string{
		"$roll",
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

func (c *Command) Run(s *discordgo.Session, m *discordgo.MessageCreate, parts []string) pkg.CommandResult {
	if len(parts) >= 2 {
		number, err := strconv.Atoi(parts[1])
		if err == nil && number >= 1 {
			v := 1 + rand.Intn(number)
			response := fmt.Sprintf("%d", v)
			utils.Reply(s, m, response)
		}
	}
	return pkg.CommandResultUserCooldown
}

func (c *Command) Description() string {
	return c.Command.Description
}
