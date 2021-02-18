package color

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/basecommand"
	"github.com/pajbot/pajbot2-discord/pkg"
	"github.com/pajbot/pajbot2-discord/pkg/commands"
	"github.com/pajbot/pajbot2-discord/pkg/utils"
)

func init() {
	commands.Register([]string{"$color", "$colour"}, New())
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

func getColorRole(c string, roles []*discordgo.Role) *discordgo.Role {
	for _, role := range roles {
		if role.Name == c {
			return role
		}
	}

	return nil
}

func (c *Command) Run(s *discordgo.Session, m *discordgo.MessageCreate, parts []string) (res pkg.CommandResult) {
	res = pkg.CommandResultUserCooldown
	const usage = `$color <colorname>`

	var err error

	hasAccess, err := utils.MemberInRoles(s, m.GuildID, m.Author.ID, "nitrobooster")
	if err != nil {
		fmt.Println("Error:", err)
		return pkg.CommandResultUserCooldown
	}
	if !hasAccess {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s, you don't have permission to use $color. Please consider boosting the server to get access to it.", m.Author.Mention()))
		return pkg.CommandResultUserCooldown
	}

	if len(parts) < 2 {
		s.ChannelMessageSend(m.ChannelID, "You need to provide a color. usage: "+usage)
		return
	}

	color := parts[1]

	if color == "reset" {
		colorRoles := utils.ColorRoles(s, m.GuildID)
		err := utils.RemoveNitroColors(s, m.GuildID, m.Author.ID, colorRoles)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Error removing nitro roles: "+err.Error())
		}

		s.ChannelMessageSend(m.ChannelID, "Removed colors from you")

		return
	}

	colorRoles := utils.ColorRoles(s, m.GuildID)

	// Ensure the color exists
	colorRole := getColorRole(color, colorRoles)
	if colorRole == nil {
		s.ChannelMessageSend(m.ChannelID, "The specified color doesn't exist. Use $colors to see what colors are available")
		return
	}

	err = utils.RemoveNitroColors(s, m.GuildID, m.Author.ID, colorRoles)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Error removing previous nitro roles: "+err.Error())
	}

	// Assign role
	err = s.GuildMemberRoleAdd(m.GuildID, m.Author.ID, colorRole.ID)
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
