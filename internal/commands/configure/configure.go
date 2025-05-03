package configure

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/basecommand"
	"github.com/pajbot/pajbot2-discord/pkg"
	"github.com/pajbot/pajbot2-discord/pkg/commands"
	"github.com/pajbot/pajbot2-discord/pkg/utils"
)

func init() {
	commands.Register([]string{"$configure", "$config", "$cfg"}, New())
}

type Command struct {
	basecommand.Command
}

func New() *Command {
	return &Command{
		Command: basecommand.New(),
	}
}

func (c *Command) Run(s *discordgo.Session, m *discordgo.MessageCreate, parts []string) (res pkg.CommandResult) {
	const usage = "usage: $configure TYPE KEY VALUE"
	res = pkg.CommandResultNoCooldown
	hasAccess, err := utils.MemberAdmin(s, m.GuildID, m.Author.ID)
	if err != nil {
		fmt.Println("Error checking perms:", err)
		return pkg.CommandResultUserCooldown
	}

	if !hasAccess {
		fmt.Println("NO PERMISSION!!!!!!!!!!")
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s, you don't have permission dummy", m.Author.Mention()))
		return pkg.CommandResultUserCooldown
	}

	// Cut off trigger
	parts = parts[1:]

	if len(parts) == 0 {
		const usage = "usage: $configure autoreact/value ..."
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s, %s", m.Author.Mention(), usage))
		return
	}

	configType := parts[0]

	// config type decides how we read the value
	switch configType {
	case "value":
		const usage = "use the /value command instead (admin only)"
		s.ChannelMessageSend(m.ChannelID, usage)

	case "autoreact":
		const usage = "use the /autoreact command instead (admin only)"
		s.ChannelMessageSend(m.ChannelID, usage)

	case "twitter":
		s.ChannelMessageSend(m.ChannelID, "twitter is dead")

	case "channel":
		const usage = "use the /channel command instead (admin only)"
		s.ChannelMessageSend(m.ChannelID, usage)

	case "role":
		const usage = "use the /role command instead (admin only)"
		s.ChannelMessageSend(m.ChannelID, usage)
	}

	return
}

func (c *Command) Description() string {
	return c.Command.Description
}
