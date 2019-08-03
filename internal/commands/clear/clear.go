package clear

import (
	"fmt"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/basecommand"
	"github.com/pajbot/pajbot2-discord/pkg"
	"github.com/pajbot/pajbot2-discord/pkg/commands"
	"github.com/pajbot/pajbot2-discord/pkg/utils"
)

func init() {
	commands.Register([]string{"$clear"}, New())
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

type Type string

const (
	TypeAll                    = "all"
	TypeHasEmbedOrAttachment   = "has:embed||has:attachment"
	TypeHasNoEmbedOrAttachment = "!has:embed&&!has:attachment"
	TypeHasAttachment          = "has:attachment"
	TypeHasEmbed               = "has:embed"
	TypeHasNoEmbed             = "!has:embed"
	TypeHasMention             = "has:mention"
)

var validTypes = []string{
	TypeAll,
	TypeHasEmbedOrAttachment,
	TypeHasNoEmbedOrAttachment,
	TypeHasAttachment,
	TypeHasEmbed,
	TypeHasNoEmbed,
	TypeHasMention,
}

func (c *Command) clear(s *discordgo.Session, m *discordgo.MessageCreate, clearCount string, predicate func(m *discordgo.Message) bool) (res pkg.CommandResult) {
	res = pkg.CommandResultNoCooldown

	var messageIDs []string

	clearCountNumber, err := strconv.Atoi(clearCount)
	if err != nil {
		fmt.Println("ERROR GETTING NUMBER:", err)
		return
	}

	if clearCountNumber > 100 {
		clearCountNumber = 100
	} else if clearCountNumber <= 0 {
		return
	}

	// Figure out what messages to delete
	messages, err := s.ChannelMessages(m.ChannelID, clearCountNumber, "", "", "")
	if err != nil {
		fmt.Println("Error getting messages", err)
		return
	}

	now := time.Now()
	oldestMessageLimit := now.Add(-time.Hour * (24 * 14))
	for _, message := range messages {
		messageTimestamp, err := message.Timestamp.Parse()
		if err != nil {
			fmt.Println("Error getting timestamp for message:", err)
			continue
		}
		if messageTimestamp.Before(oldestMessageLimit) {
			// message too old, skipping it
			continue
		}
		// Skip messages that are too old
		// DO FILTERING
		if predicate(message) {
			messageIDs = append(messageIDs, message.ID)
		}
	}

	// Bulk delete 100 at a time
	err = utils.DeleteChunks(s, m.ChannelID, messageIDs)
	if err != nil {
		fmt.Println("ERROR DELETING MESSAGES:", err)
		return
	}

	return
}

func (c *Command) Run(s *discordgo.Session, m *discordgo.MessageCreate, parts []string) (res pkg.CommandResult) {
	res = pkg.CommandResultNoCooldown
	const commandName = "$clear"
	const usage = `$clear type searchLength`

	var err error

	hasAccess, err := utils.MemberAdmin(s, m.GuildID, m.Author.ID)
	if err != nil {
		fmt.Println("Error:", err)
		return pkg.CommandResultUserCooldown
	}
	if !hasAccess {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s, you don't have permission to use "+commandName, m.Author.Mention()))
		return pkg.CommandResultUserCooldown
	}

	parts = parts[1:]

	if len(parts) < 2 {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s, usage: %s. valid types: %s", m.Author.Mention(), usage, validTypes))
		return
	}

	clearType := parts[0]
	clearCount := parts[1]

	const unimplementedTypeFormat = "%s, unimplemented clear type %s"

	switch clearType {
	case TypeAll:
		res = c.clear(s, m, clearCount, func(m *discordgo.Message) bool {
			return true
		})
	case TypeHasEmbedOrAttachment:
		res = c.clear(s, m, clearCount, func(m *discordgo.Message) bool {
			return len(m.Embeds) > 0 || len(m.Attachments) > 0
		})
	case TypeHasNoEmbedOrAttachment:
		res = c.clear(s, m, clearCount, func(m *discordgo.Message) bool {
			return len(m.Embeds) == 0 && len(m.Attachments) == 0
		})
	case TypeHasAttachment:
		res = c.clear(s, m, clearCount, func(m *discordgo.Message) bool {
			return len(m.Attachments) > 0
		})
	case TypeHasEmbed:
		res = c.clear(s, m, clearCount, func(m *discordgo.Message) bool {
			return len(m.Embeds) > 0
		})
	case TypeHasNoEmbed:
		res = c.clear(s, m, clearCount, func(m *discordgo.Message) bool {
			return len(m.Embeds) == 0
		})
	case TypeHasMention:
		res = c.clear(s, m, clearCount, func(m *discordgo.Message) bool {
			return len(m.Mentions) > 0
		})
	default:
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf(unimplementedTypeFormat, m.Author.Mention(), clearType))
	}

	return
}

func (c *Command) Description() string {
	return c.Command.Description
}
