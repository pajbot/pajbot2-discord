package ban

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/basecommand"
	"github.com/pajbot/pajbot2-discord/internal/serverconfig"
	"github.com/pajbot/pajbot2-discord/pkg"
	"github.com/pajbot/pajbot2-discord/pkg/commands"
	"github.com/pajbot/pajbot2-discord/pkg/utils"
)

func init() {
	commands.Register([]string{"$ban", "$anonban"}, New())
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
	res = pkg.CommandResultNoCooldown
	hasAccess, err := utils.MemberInRoles(s, m.GuildID, m.Author.ID, "mod")
	if err != nil {
		fmt.Println("Error:", err)
		return pkg.CommandResultUserCooldown
	}

	if !hasAccess {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s, you don't have permission dummy", m.Author.Mention()))
		return pkg.CommandResultUserCooldown
	}

	// $ban or $anonban
	commandName := parts[0]

	if len(m.Mentions) == 0 {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("missing user arg. usage: %s <user> <reason>", commandName))
		return
	}

	target := m.Mentions[0]

	if len(parts) < 3 {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("missing reason arg. usage: %s <user> <reason>", commandName))
		return
	}

	reason := strings.Join(parts[2:], " ")

	targetChannel := serverconfig.Get(m.GuildID, "channel:moderation-action")
	if targetChannel == "" {
		fmt.Println("No channel set up for moderation actions")
		return
	}

	isAnonBan := commandName == "$anonban"

	resultMessage := ""
	if isAnonBan {
		const resultFormat = "Banning %s for reason: %s"
		resultMessage = fmt.Sprintf(resultFormat, utils.MentionUser(s, m.GuildID, target), reason)
	} else {
		const resultFormat = "%s banned %s for reason: %s"
		resultMessage = fmt.Sprintf(resultFormat, utils.MentionUser(s, m.GuildID, m.Author), utils.MentionUser(s, m.GuildID, target), reason)
	}

	s.ChannelMessageSend(m.ChannelID, resultMessage)
	s.ChannelMessageSend(targetChannel, resultMessage)
	s.GuildBanCreateWithReason(m.GuildID, target.ID, reason, 0)

	return
}

func (c *Command) Description() string {
	return c.Command.Description
}
