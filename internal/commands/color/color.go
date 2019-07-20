package color

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/basecommand"
	"github.com/pajbot/pajbot2-discord/internal/config"
	"github.com/pajbot/pajbot2-discord/pkg"
	"github.com/pajbot/pajbot2-discord/pkg/commands"
	"github.com/pajbot/pajbot2-discord/pkg/utils"
)

func init() {
	commands.Register([]string{"$color"}, New())
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

func colorIsValid(c string){
	for _, b := range config.ColorPickerNames {
        if b == c {
            return true
        }
    }
    return false
}

func (c *Command) Run(s *discordgo.Session, m *discordgo.MessageCreate, parts []string) (res pkg.CommandResult) {
	res = pkg.CommandResultUserCooldown
	const usage = `$color <colorname>`
	
	var err error
	var role string
	
	hasAccess, err := utils.MemberInRoles(s, m.GuildID, m.Author.ID, config.ColorPickerRoles)
	if err != nil {
		fmt.Println("Error:", err)
		return pkg.CommandResultUserCooldown
	}
	if !hasAccess {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s, you don't have permission to use $color. Please consider boosting the server to get access to it.", m.Author.Mention()))
		return pkg.CommandResultUserCooldown
	}
	
	color := parts[1]
	
	if len(parts) < 1 {
		s.ChannelMessageSend(m.ChannelID, "You need to provide a valid color. usage: "+usage)
		return
	}
	
	// Check that role exists
	if !colorIsValid(color) {
		s.ChannelMessageSend(m.ChannelID, "The specified color doesn't exist. Use $colors to see what colors are available")
		return
	}

	// Assign role
	err = s.GuildMemberRoleAdd(m.GuildID, targetID, config.ColorPickerRoleMap[color])
	if err != nil {
		fmt.Println("Error assigning role:", err)
		s.ChannelMessageSend(m.ChannelID, "Couldn't apply color correctly. Ping @pajlada for more info.")
	}

	// Announce color success
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Color %s assigned correctly.", color))

	return
}

func (c *Command) Description() string {
	return c.Command.Description
}
