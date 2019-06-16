package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pajlada/pajbot2-discord/internal/config"
	"github.com/pajlada/pajbot2-discord/pkg"
	"github.com/pajlada/pajbot2-discord/pkg/commands"
	"github.com/pajlada/stupidmigration"

	_ "github.com/lib/pq"

	_ "github.com/pajlada/pajbot2-discord/internal/commands/ban"
	_ "github.com/pajlada/pajbot2-discord/internal/commands/channelinfo"
	_ "github.com/pajlada/pajbot2-discord/internal/commands/channels"
	_ "github.com/pajlada/pajbot2-discord/internal/commands/guildinfo"
	_ "github.com/pajlada/pajbot2-discord/internal/commands/modcommands"
	_ "github.com/pajlada/pajbot2-discord/internal/commands/mute"
	_ "github.com/pajlada/pajbot2-discord/internal/commands/ping"
	_ "github.com/pajlada/pajbot2-discord/internal/commands/roleinfo"
	_ "github.com/pajlada/pajbot2-discord/internal/commands/roles"
	_ "github.com/pajlada/pajbot2-discord/internal/commands/tags"
	_ "github.com/pajlada/pajbot2-discord/internal/commands/userid"
)

var sqlClient *sql.DB

func init() {
	var err error
	sqlClient, err = sql.Open("postgres", config.DSN)
	if err != nil {
		fmt.Println("Unable to connect to mysql", err)
		os.Exit(1)
	}

	err = sqlClient.Ping()
	if err != nil {
		fmt.Println("Unable to ping mysql", err)
		os.Exit(1)
	}

	err = stupidmigration.Migrate("migrations", sqlClient)
	if err != nil {
		fmt.Println("Unable to run SQL migrations", err)
		os.Exit(1)
	}

	commands.SQLClient = sqlClient
}

func main() {
	bot, err := discordgo.New("Bot " + config.Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	go func() {
		for {
			<-time.After(3 * time.Second)
			now := time.Now()
			const query = `SELECT id, action, timepoint FROM discord_queue ORDER BY timepoint DESC LIMIT 30;`
			rows, err := sqlClient.Query(query)
			if err != nil {
				fmt.Println("err:", err)
				continue
			}
			var actionsToRemove []int64
			defer rows.Close()
			for rows.Next() {
				var (
					id           int64
					actionString string
					timepoint    time.Time
				)
				if err := rows.Scan(&id, &actionString, &timepoint); err != nil {
					fmt.Println("Error scanning:", err)
				}
				if timepoint.After(now) {
					continue
				}

				var action pkg.Action
				err = json.Unmarshal([]byte(actionString), &action)
				if err != nil {
					fmt.Println("Error unmarshaling action:", err)
					continue
				}
				fmt.Println("Perform", action.Type)

				switch action.Type {
				case "unmute":
					err = bot.GuildMemberRoleRemove(action.GuildID, action.UserID, action.RoleID)
					if err != nil {
						fmt.Println("Error removing role")
						continue
					}

					actionsToRemove = append(actionsToRemove, id)
				}
			}

			for _, actionID := range actionsToRemove {
				sqlClient.Exec("DELETE FROM discord_queue WHERE id=$1", actionID)
			}
		}
	}()

	bot.AddHandler(onMessage)
	bot.AddHandler(onMessageDeleted)
	bot.AddHandler(onUserBanned)
	bot.AddHandler(onMessageReactionAdded)
	bot.AddHandler(onMessageReactionRemoved)

	// Open a websocket connection to Discord and begin listening.
	err = bot.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	defer bot.Close()

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

func pushMessageIntoDatabase(m *discordgo.MessageCreate) (err error) {
	const query = `INSERT INTO discord_messages (id, content) VALUES ($1, $2)`
	_, err = sqlClient.Exec(query, m.ID, m.Content)
	return
}

func getMessageFromDatabase(messageID string) (content string, err error) {
	const query = `SELECT content FROM discord_messages WHERE id=$1`
	row := sqlClient.QueryRow(query, messageID)
	err = row.Scan(&content)
	return
}

func onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	// push message into database
	err := pushMessageIntoDatabase(m)
	if err != nil {
		log.Println("Error pushing message into databasE:", err)
	}

	if m.Author.ID == s.State.User.ID {
		return
	}

	c, parts := commands.Match(m.Content)
	if c != nil {
		if cmd, ok := c.(pkg.Command); ok {
			id := m.ChannelID + m.Author.ID
			if cmd.HasUserIDCooldown(id) {
				return
			}

			switch cmd.Run(s, m, parts) {
			case pkg.CommandResultUserCooldown:
				cmd.AddUserIDCooldown(id)
			case pkg.CommandResultGlobalCooldown:
				cmd.AddGlobalCooldown()
			case pkg.CommandResultFullCooldown:
				cmd.AddUserIDCooldown(id)
				cmd.AddGlobalCooldown()
			}
		} else if f, ok := c.(func(s *discordgo.Session, m *discordgo.MessageCreate, parts []string)); ok {
			f(s, m, parts)
		}
	}
}

func onMessageDeleted(s *discordgo.Session, m *discordgo.MessageDelete) {
	var output string
	messageContent, err := getMessageFromDatabase(m.ID)
	if err != nil {
		fmt.Println("Error getting full message")
	}
	output += "Message deleted in <#" + m.ChannelID + ">"
	output += "\nContent: `" + messageContent + "`"
	if m.Author != nil {
		output += "\nAuthor: <@" + m.Author.ID + "> (" + m.Author.Username + " (" + m.Author.ID + "))"
	}
	s.ChannelMessageSend(config.ActionLogChannelID, output)
}

func onUserBanned(s *discordgo.Session, m *discordgo.GuildBanAdd) {
	auditLog, err := s.GuildAuditLog(m.GuildID, "", "", 22, 1)
	if err != nil {
		fmt.Println("Error getting user ban data", err)
		return
	}
	fmt.Println(auditLog)
	if len(auditLog.AuditLogEntries) != 1 {
		fmt.Println("Unable to get the single ban entry")
		return
	}
	if len(auditLog.Users) != 2 {
		fmt.Println("length of users is wrong")
		return
	}
	banner := auditLog.Users[0]
	bannedUser := auditLog.Users[1]
	if bannedUser.ID != m.User.ID {
		fmt.Println("got log for wrong use Pepega")
		return
	}
	fmt.Println(auditLog.Users)
	entry := auditLog.AuditLogEntries[0]
	// var username string
	// for _ user := range auditLog.Users {
	// 	if user.ID == entry.
	// }
	fmt.Println(entry)
	fmt.Println("Entry User ID:", entry.UserID)
	fmt.Println("target user ID:", m.User.ID)
	s.ChannelMessageSend(config.ModerationActionChannelID, fmt.Sprintf("%s was banned by %s: %s", m.User.Mention(), banner.Username, entry.Reason))
}

// const weebMessageID = `552788256333234176`
const weebMessageID = `552791672854151190`
const reactionBye = "👋"

func onMessageReactionAdded(s *discordgo.Session, m *discordgo.MessageReactionAdd) {
	if m.MessageID == weebMessageID {
		if m.Emoji.Name == reactionBye {
			c, err := s.State.Channel(config.WeebChannelID)
			if err != nil {
				fmt.Println("err:", err)
				return
			}
			var overwriteDenies int
			for _, overwrite := range c.PermissionOverwrites {
				if overwrite.Type == "member" && overwrite.ID == m.UserID {
					overwriteDenies = overwrite.Deny
				}
			}
			if overwriteDenies != 0 {
				// s.ChannelMessageSend(m.ChannelID, "cannot set your permissions - you have weird permissions set from before")
				return
			}

			err = s.ChannelPermissionSet(config.WeebChannelID, m.UserID, "member", 0, discordgo.PermissionReadMessages)
			if err != nil {
				fmt.Println("uh oh something went wrong")
				return
			}

			// s.ChannelMessageSend(m.ChannelID, "added permission")
		}
	}
}

func onMessageReactionRemoved(s *discordgo.Session, m *discordgo.MessageReactionRemove) {
	if m.MessageID == weebMessageID {
		if m.Emoji.Name == reactionBye {
			c, err := s.State.Channel(config.WeebChannelID)
			if err != nil {
				fmt.Println("err:", err)
				return
			}
			var overwriteDenies int
			for _, overwrite := range c.PermissionOverwrites {
				if overwrite.Type == "member" && overwrite.ID == m.UserID {
					overwriteDenies = overwrite.Deny
				}
			}

			if overwriteDenies != discordgo.PermissionReadMessages {
				// s.ChannelMessageSend(m.ChannelID, "not allowed to remove that permission buddy")
				return
			}

			err = s.ChannelPermissionDelete(config.WeebChannelID, m.UserID)
			if err != nil {
				fmt.Println("uh oh something went wrong")
				return
			}
			// s.ChannelMessageSend(m.ChannelID, "removed permission")
		}
	}
}
