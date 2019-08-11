package configure

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/basecommand"
	"github.com/pajbot/pajbot2-discord/internal/channels"
	"github.com/pajbot/pajbot2-discord/internal/roles"
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

func (c *Command) configureTwitter(s *discordgo.Session, m *discordgo.MessageCreate, parts []string) {
	const usage = "usage: $configure twitter KEY(username) VALUE"
	if len(parts) < 3 {
		s.ChannelMessageSend(m.ChannelID, usage)
		return
	}
	key := parts[1]
	value := parts[2]

	switch key {
	case "username":
	default:
		s.ChannelMessageSend(m.ChannelID, "Invalid key argument. "+usage)
		return
	}

	key = "twitter:" + key

	err := serverconfig.Save(commands.SQLClient, m.GuildID, key, value)
	if err != nil {
		log.Println("SQL Error in set:", err)
		return
	}
}

func (c *Command) Run(s *discordgo.Session, m *discordgo.MessageCreate, parts []string) (res pkg.CommandResult) {
	const usage = "usage: $configure TYPE KEY VALUE"
	res = pkg.CommandResultNoCooldown
	fmt.Println("a")
	hasAccess, err := utils.MemberAdmin(s, m.GuildID, m.Author.ID)
	fmt.Println("b")
	if err != nil {
		fmt.Println("Error checking perms:", err)
		return pkg.CommandResultUserCooldown
	}

	if !hasAccess {
		fmt.Println("NO PERMISSION!!!!!!!!!!")
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s, you don't have permission dummy", m.Author.Mention()))
		return pkg.CommandResultUserCooldown
	}

	parts = parts[1:]

	var configType, key, value string
	configType = parts[0]

	// config type decides how we read the value
	switch configType {
	case "twitter":
		c.configureTwitter(s, m, parts)

	case "channel":
		const usage = "usage: $configure channel CHANNEL_ROLE VALUE(here/reset/get)"
		if len(parts) < 3 {
			s.ChannelMessageSend(m.ChannelID, usage)
			return
		}
		key = parts[1]
		value = parts[2]

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
			err := serverconfig.Save(commands.SQLClient, m.GuildID, key, newChannelID)
			if err != nil {
				log.Println("SQL Error in set:", err)
				return
			}
		} else {
			err := serverconfig.Remove(commands.SQLClient, m.GuildID, key)
			if err != nil {
				log.Println("SQL Error in remove:", err)
				return
			}
		}

	case "role":
		const usage = "usage: $configure role RoleName(minimod/mod/admin/muted/nitrobooster) ServerRoleID(e.g. 598492056981)"
		if len(parts) < 3 {
			s.ChannelMessageSend(m.ChannelID, usage)
			return
		}
		key = parts[1]
		value = parts[2]

		if !roles.Valid(key) {
			s.ChannelMessageSend(m.ChannelID, "Invalid key argument. "+usage)
			return
		}

		key = configType + ":" + key

		roles, err := s.GuildRoles(m.GuildID)
		if err != nil {
			const f = "%s, error getting guild roles: %s"
			r := fmt.Sprintf(f, m.Author.Mention(), err)
			s.ChannelMessageSend(m.ChannelID, r)

			return
		}

		oldValue := serverconfig.Get(m.GuildID, key)

		switch value {
		case "reset":
			err := serverconfig.Remove(commands.SQLClient, m.GuildID, key)
			if err != nil {
				const f = "%s, error removing role: %s"
				r := fmt.Sprintf(f, m.Author.Mention(), err)
				s.ChannelMessageSend(m.ChannelID, r)

				return
			}

			const f = "%s, key %s reset. old value was %s"
			r := fmt.Sprintf(f, m.Author.Mention(), key, oldValue)
			s.ChannelMessageSend(m.ChannelID, r)

			return

		case "get":
			const f = "%s, value of %s is %s"
			r := fmt.Sprintf(f, m.Author.Mention(), key, oldValue)
			s.ChannelMessageSend(m.ChannelID, r)
			return

		default:
			var serverRole *discordgo.Role

			for _, r := range roles {
				if r.ID == value {
					serverRole = r
					continue
				}
			}

			if serverRole == nil {
				const f = "%s, no role found with the id '%s'. Use $roles to list all available roles"
				r := fmt.Sprintf(f, m.Author.Mention(), value)
				s.ChannelMessageSend(m.ChannelID, r)

				return
			}

			err := serverconfig.Save(commands.SQLClient, m.GuildID, key, serverRole.ID)
			if err != nil {
				const f = "%s, error saving role: %s"
				r := fmt.Sprintf(f, m.Author.Mention(), err)
				s.ChannelMessageSend(m.ChannelID, r)

				return
			}

			const f = "%s, new role for %s is %s"
			r := fmt.Sprintf(f, m.Author.Mention(), key, serverRole.ID)
			s.ChannelMessageSend(m.ChannelID, r)

			return
		}
	}

	return
}

func (c *Command) Description() string {
	return c.Command.Description
}
