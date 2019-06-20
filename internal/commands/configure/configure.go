package configure

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/basecommand"
	"github.com/pajbot/pajbot2-discord/internal/channels"
	"github.com/pajbot/pajbot2-discord/internal/config"
	"github.com/pajbot/pajbot2-discord/internal/serverconfig"
	"github.com/pajbot/pajbot2-discord/pkg"
	"github.com/pajbot/pajbot2-discord/pkg/commands"
	"github.com/pajbot/pajbot2-discord/pkg/utils"
)

func init() {
	commands.Register([]string{"$configure"}, New())
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
	hasAccess, err := utils.MemberInRoles(s, m.GuildID, m.Author.ID, config.AdminRoles)
	if err != nil {
		return pkg.CommandResultUserCooldown
	}

	if !hasAccess {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s, you don't have permission dummy", m.Author.Mention()))
		return pkg.CommandResultUserCooldown
	}

	parts = parts[1:]

	if len(parts) < 3 {
		s.ChannelMessageSend(m.ChannelID, usage)
		return
	}

	var configType, key, value string
	configType = parts[0]
	key = parts[1]
	value = parts[2]

	// config type decides how we read the value
	switch configType {
	case "channel":
		const usage = "usage: $configure channel CHANNEL_ROLE VALUE(here/reset/get)"

		if !channels.ValidRole(key) {
			s.ChannelMessageSend(m.ChannelID, "Invalid key argument. "+usage)
			return
		}

		key = configType + ":" + key

		oldChannelID := serverconfig.Get(m.GuildID, key)

		var newChannelID string

		switch value {
		case "here":
			s.ChannelMessageSend(m.ChannelID, "channel here. old: "+oldChannelID)
			newChannelID = m.ChannelID
		case "reset":
			s.ChannelMessageSend(m.ChannelID, "channel reset. old: "+oldChannelID)
			newChannelID = ""
		case "get":
			s.ChannelMessageSend(m.ChannelID, "Current: "+oldChannelID+". <#"+oldChannelID+">")
			return
		default:
			s.ChannelMessageSend(m.ChannelID, usage)
			return
		}

		if newChannelID != "" {
			err := set(m.GuildID, key, newChannelID)
			if err != nil {
				log.Println("SQL Error in set:", err)
				return
			}
		} else {
			err := remove(m.GuildID, key)
			if err != nil {
				log.Println("SQL Error in remove:", err)
				return
			}
		}

		serverconfig.Set(m.GuildID, key, newChannelID)
	}

	return
}

func (c *Command) Description() string {
	return c.Command.Description
}

func set(serverID, key, value string) error {
	query := "INSERT INTO config (server_id, key, value) VALUES ($1, $2, $3) ON CONFLICT (server_id, key) DO UPDATE SET value=$3"
	_, err := commands.SQLClient.Exec(query, serverID, key, value)
	return err
}

func remove(serverID, key string) error {
	query := "DELETE FROM config WHERE server_id=$1 AND key=$2"
	_, err := commands.SQLClient.Exec(query, serverID, key)
	return err
}
