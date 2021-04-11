package ban

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jessevdk/go-flags"
	"github.com/pajbot/basecommand"
	"github.com/pajbot/pajbot2-discord/internal/serverconfig"
	"github.com/pajbot/pajbot2-discord/pkg"
	"github.com/pajbot/pajbot2-discord/pkg/commands"
	"github.com/pajbot/pajbot2-discord/pkg/utils"
)

func init() {
	commands.Register([]string{"$multiban"}, NewMultiBan())
}

type MultiBan struct {
	basecommand.Command
}

func NewMultiBan() *MultiBan {
	return &MultiBan{
		Command: basecommand.New(),
	}
}

type options struct {
	DryDun      bool   `long:"dry-run"`
	CreatedFrom string `long:"created-from" required:"true"`
	CreatedTo   string `long:"created-to" required:"true"`
}

const layout = "2006-01-02T15:04:05"
const dankCircle = "a:dankCircle:629327595216830505"
const doneEmoji = "✅"

func (c *MultiBan) Run(s *discordgo.Session, m *discordgo.MessageCreate, parts []string) (res pkg.CommandResult) {
	var opts options

	err := s.MessageReactionAdd(m.ChannelID, m.ID, dankCircle)
	if err != nil {
		fmt.Println("Error adding reaction:", err)
	}

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

	parts = parts[1:]

	args, err := flags.ParseArgs(&opts, parts)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s, error parsing flags: %s", m.Author.Mention(), err.Error()))
		return
	}

	fmt.Println("Created from:", opts.CreatedFrom)
	fmt.Println("Created to:", opts.CreatedTo)

	createdFrom, err := time.Parse(layout, opts.CreatedFrom)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s, error parsing --created-from parameter '%s'. Format must be like this: --created-from='%s': %s", m.Author.Mention(), opts.CreatedFrom, layout, err.Error()))
		return
	}

	createdTo, err := time.Parse(layout, opts.CreatedTo)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s, error parsing --created-to parameter '%s'. Format must be like this: --created-to='%s': %s", m.Author.Mention(), opts.CreatedTo, layout, err.Error()))
		return
	}

	defer func() {
		botUser, err := s.User("@me")
		if err != nil {
			fmt.Println("cant get me!")
			return
		}

		err = s.MessageReactionRemove(m.ChannelID, m.ID, dankCircle, botUser.ID)
		if err != nil {
			fmt.Println("Error removing reaction:", err)
		}
		err = s.MessageReactionAdd(m.ChannelID, m.ID, doneEmoji)
		if err != nil {
			fmt.Println("Error adding done reaction:", err)
		}
	}()

	reason := strings.Join(args, " ")

	const limit = 1000
	after := "0"

	var targets []*discordgo.Member

	for {
		members, err := s.GuildMembers(m.GuildID, after, limit)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s, error getting guild members: %s", m.Author.Mention(), err.Error()))
			return
		}

		fmt.Println("Processing", len(members), "members")

		if len(members) == 0 {
			break
		}

		for _, member := range members {
			accountCreationDate, err := discordgo.SnowflakeTimestamp(member.User.ID)
			if err != nil {
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s, error getting account creation date: %s", m.Author.Mention(), err.Error()))
				return
			}

			if accountCreationDate.After(createdFrom) && accountCreationDate.Before(createdTo) {
				targets = append(targets, member)
			}

			after = member.User.ID
		}
	}

	if len(targets) == 0 {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s, no targets found with this description", m.Author.Mention()))
		return
	}

	targetChannel := serverconfig.Get(m.GuildID, "channel:moderation-action")
	if targetChannel == "" {
		fmt.Println("No channel set up for moderation actions")
		return
	}

	// XXX
	targetChannel = m.ChannelID

	var chunks []string
	for _, target := range targets {
		fmt.Println("Target:", target.User.Username)
		chunks = append(chunks, fmt.Sprintf("%s(%s)", target.User.Username, target.User.ID))
	}

	utils.SendChunks(fmt.Sprintf("Users found matching options: Creation Date >= %s && Creation Date <= %s", opts.CreatedFrom, opts.CreatedTo), "", chunks, targetChannel, s)

	s.ChannelMessageSend(m.ChannelID, reason)
	// s.ChannelMessageSend(targetChannel, resultMessage)
	// s.GuildBanCreateWithReason(m.GuildID, target.ID, reason, 0)

	return
}

func (c *MultiBan) Description() string {
	return c.Command.Description
}
