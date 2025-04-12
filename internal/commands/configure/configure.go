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

// configureValue configures a generic value to be used elsewhere in the code
// it's for values that might not make sense to have in a separate category (like channels)
func (c *Command) configureValue(s *discordgo.Session, m *discordgo.MessageCreate, parts []string) {
	validKeys := map[string]string{
		"pajbot_host": "The host used for any pajbot API requests and commands (like $points). Example value: forsen.tv. Assumes https schema and standard api path.",

		"member_role_mode": "0 = dont use, 1 = auto grant, 2 = require twitch verification",

		"stream_ids": "List of comma-separated stream IDs to announce",
	}

	const usage = "usage: $configure value (reset/set/get/keys) [key [value]]"
	if len(parts) < 2 {
		s.ChannelMessageSend(m.ChannelID, usage)
		return
	}

	if parts[1] == "keys" {
		var chunks []string
		for validKey, keyDescription := range validKeys {
			chunks = append(chunks, fmt.Sprintf("%s: %s\n", validKey, keyDescription))
		}
		utils.SendChunks("Valid keys:\n```", "```", chunks, m.ChannelID, s)

		return
	}

	if len(parts) < 3 {
		s.ChannelMessageSend(m.ChannelID, usage)
		return
	}

	key := parts[2]

	if _, ok := validKeys[key]; !ok {
		const f = "%s, %s is not a valid key. use $configure value keys to see valid keys"
		r := fmt.Sprintf(f, m.Author.Mention(), key)
		s.ChannelMessageSend(m.ChannelID, r)
		return
	}

	dbKey := fmt.Sprintf("value:%s", key)

	oldValue := serverconfig.Get(m.GuildID, dbKey)

	switch parts[1] {
	case "reset":
		err := serverconfig.Remove(commands.SQLClient, m.GuildID, dbKey)
		if err != nil {
			fmt.Println("Error removing value:", err)
			return
		}

		var r string
		if oldValue == "" {
			const f = "%s, key %s reset"
			r = fmt.Sprintf(f, m.Author.Mention(), key)
		} else {
			const f = "%s, key %s reset. old value was %s"
			r = fmt.Sprintf(f, m.Author.Mention(), key, oldValue)
		}
		s.ChannelMessageSend(m.ChannelID, r)

		return

	case "get":
		if oldValue == "" {
			const f = "%s, key %s has no value"
			r := fmt.Sprintf(f, m.Author.Mention(), key)
			s.ChannelMessageSend(m.ChannelID, r)

			return
		}

		const f = "%s, key %s has the value %s"
		r := fmt.Sprintf(f, m.Author.Mention(), key, oldValue)
		s.ChannelMessageSend(m.ChannelID, r)

		return

	case "set":
		const usage = "usage: $configure value set key value"
		if len(parts) < 4 {
			s.ChannelMessageSend(m.ChannelID, usage)
			return
		}

		value := strings.Join(parts[3:], " ")

		if len(value) == 0 {
			s.ChannelMessageSend(m.ChannelID, "No valid value was set, use reset to remove the set value")
			return
		}

		err := serverconfig.Save(commands.SQLClient, m.GuildID, dbKey, value)
		if err != nil {
			log.Println("SQL Error in set:", err)
			return
		}

		var r string
		if oldValue == "" {
			const f = "%s, new value for key %s is %s"
			r = fmt.Sprintf(f, m.Author.Mention(), key, value)
		} else {
			const f = "%s, new value for key %s is %s (old value was %s)"
			r = fmt.Sprintf(f, m.Author.Mention(), key, value, oldValue)
		}
		s.ChannelMessageSend(m.ChannelID, r)

		return

	default:
		s.ChannelMessageSend(m.ChannelID, "Invalid key argument. "+usage)
		return
	}
}

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
		// Configure a generic value
		c.configureValue(s, m, parts)

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
