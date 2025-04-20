package configure

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/basecommand"
	"github.com/pajbot/pajbot2-discord/internal/serverconfig"
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

var discordEmojiRegex = regexp.MustCompile(`<(a)?:([^<>:]+):([0-9]+)>`)

var unicodeEmojiRegex = regexp.MustCompile(`[\x{00A0}-\x{1F9EF}]`)

func (c *Command) configureAutoReact(s *discordgo.Session, m *discordgo.MessageCreate, parts []string) {
	const usage = "usage: $configure autoreact (reset/set/get) EMOJIS..."
	if len(parts) < 2 {
		s.ChannelMessageSend(m.ChannelID, usage)
		return
	}
	key := fmt.Sprintf("autoreact:%s", m.ChannelID)

	switch parts[1] {
	case "reset":
		err := serverconfig.Remove(commands.SQLClient, m.GuildID, key)
		if err != nil {
			fmt.Println("error removing shit:", err)
		}

		return

	case "get":
		emojis := serverconfig.Get(m.GuildID, key)
		s.ChannelMessageSend(m.ChannelID, "Current autoreact IDs: "+emojis)

		return

	case "set":
		if len(parts) < 2 {
			s.ChannelMessageSend(m.ChannelID, usage)
			return
		}

		remainder := strings.Join(parts[2:], " ")

		emojiIDs := unicodeEmojiRegex.FindAllString(remainder, -1)
		discordEmojis := discordEmojiRegex.FindAllStringSubmatch(remainder, -1)

		for _, discordEmoji := range discordEmojis {
			emojiIDs = append(emojiIDs, fmt.Sprintf("%s:%s", discordEmoji[2], discordEmoji[3]))
			// emojiIDs = append(emojiIDs, discordEmoji[0])
		}

		if len(emojiIDs) == 0 {
			s.ChannelMessageSend(m.ChannelID, "no valid emojis passed to the set function 😕")
			return
		}

		emojiIDsString := strings.Join(emojiIDs, ",")

		s.ChannelMessageSend(m.ChannelID, "Set autoreact to: "+emojiIDsString)

		err := serverconfig.Save(commands.SQLClient, m.GuildID, key, emojiIDsString)
		if err != nil {
			log.Println("SQL Error in set:", err)
			return
		}

		return

	default:
		s.ChannelMessageSend(m.ChannelID, "Invalid key argument. "+usage)
		return
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
		c.configureAutoReact(s, m, parts)

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
