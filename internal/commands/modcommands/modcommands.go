package modcommands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/basecommand"
	"github.com/pajbot/pajbot2-discord/pkg"
	"github.com/pajbot/pajbot2-discord/pkg/commands"
)

func init() {
	commands.Register([]string{"$modcommands"}, New())
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
	// hasAccess, err := utils.MemberHasPermission(s, m.GuildID, m.Author.ID, "minimod")
	// if err != nil {
	// 	fmt.Println("Error:", err)
	// 	return
	// }
	// if !hasAccess {
	// 	// No permission
	// 	return
	// }

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
	return c.Command.Description
}
