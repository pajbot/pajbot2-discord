package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/pajbot2-discord/internal/config"
	"github.com/pajbot/pajbot2-discord/internal/mute"
	"github.com/pajbot/pajbot2-discord/internal/serverconfig"
	"github.com/pajbot/pajbot2-discord/pkg"
	"github.com/pajbot/pajbot2-discord/pkg/commands"
	"github.com/pajlada/stupidmigration"

	_ "github.com/lib/pq"

	_ "github.com/pajbot/pajbot2-discord/internal/commands/ban"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/channelinfo"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/channels"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/color"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/colors"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/configure"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/guildinfo"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/modcommands"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/mute"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/ping"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/points"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/roleinfo"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/roles"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/tags"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/userid"
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

	// Load channel roles from config
	const query = "SELECT server_id, key, value FROM config"
	rows, err := sqlClient.Query(query)
	if err != nil {
		fmt.Println("Error loading channel roles:", err)
		os.Exit(1)
	}

	for rows.Next() {
		var serverID, key, value string
		err := rows.Scan(&serverID, &key, &value)
		if err != nil {
			fmt.Println("Error scanning channel roles:", err)
			os.Exit(1)
		}

		serverconfig.Set(serverID, key, value)
	}
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
	bot.AddHandler(onPresenceUpdate)
	bot.AddHandler(func(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
		onMemberJoin(s, m, sqlClient)
	})

	// Open a websocket connection to Discord and begin listening.
	err = bot.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	defer bot.Close()

	go func() {
		const resultFormat = "%s was unmuted (reason was %s)"
		for {
			<-time.After(3 * time.Second)
			unmutedUsers, err := mute.ExpireMutes(bot, sqlClient)
			if err != nil {
				fmt.Println("err:", err)
			}

			for _, unmutedUser := range unmutedUsers {
				member, err := bot.GuildMember(unmutedUser.GuildID, unmutedUser.UserID)
				if err != nil {
					fmt.Println("Error getting guild member:", err)
					continue
				}
				resultMessage := fmt.Sprintf(resultFormat, member.Mention(), unmutedUser.Reason)
				targetChannel := serverconfig.Get(unmutedUser.GuildID, "channel:moderation-action")
				if targetChannel == "" {
					fmt.Println("No channel set up for moderation actions")
					return
				}
				bot.ChannelMessageSend(targetChannel, resultMessage)
			}
		}
	}()

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

func pushMessageIntoDatabase(m *discordgo.MessageCreate) (err error) {
	const query = `INSERT INTO discord_messages (id, content, author_id) VALUES ($1, $2, $3)`
	authorID := "unknown"
	if m.Author != nil {
		authorID = m.Author.ID
	}
	_, err = sqlClient.Exec(query, m.ID, m.Content, authorID)
	return
}

func getMessageFromDatabase(messageID string) (content string, authorID string, err error) {
	const query = `SELECT content, author_id FROM discord_messages WHERE id=$1`
	row := sqlClient.QueryRow(query, messageID)
	err = row.Scan(&content, &authorID)
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

// TODO: Don't list messages deleted from or by ourselves (this bot account)
func onMessageDeleted(s *discordgo.Session, m *discordgo.MessageDelete) {
	var authorID string
	messageContent, authorID, err := getMessageFromDatabase(m.ID)
	if err != nil {
		fmt.Println("Error getting full message")
	}

	targetChannel := serverconfig.Get(m.GuildID, "channel:action-log")
	if targetChannel == "" {
		fmt.Println("No channel set up for action log")
		return
	}

	// Try to get member
	var member *discordgo.Member
	if authorID != "unknown" {
		member, err = s.GuildMember(m.GuildID, authorID)
		if err != nil {
			fmt.Println("Error getting guild member:", err)
		}
	}

	embed := &discordgo.MessageEmbed{
		Title: "Message deleted",
	}

	if member != nil {
		payload := fmt.Sprintf("<@%s> - Name: %s#%s - ID: %s", authorID, member.User.Username, member.User.Discriminator, authorID)
		if member.Nick != "" {
			payload += " Nickname: " + member.Nick
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Author",
			Value:  payload,
			Inline: true,
		})
	} else {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Author",
			Value:  "unknown",
			Inline: true,
		})
	}
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "Content",
		Value:  strings.Replace(messageContent, "`", "", -1),
		Inline: true,
	})
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "Channel",
		Value:  "<#" + m.ChannelID + ">",
		Inline: true,
	})
	s.ChannelMessageSendEmbed(targetChannel, embed)
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

	targetChannel := serverconfig.Get(m.GuildID, "channel:moderation-action")
	if targetChannel == "" {
		fmt.Println("No channel set up for moderation actions")
		return
	}

	s.ChannelMessageSend(targetChannel, fmt.Sprintf("%s was banned by %s: %s", m.User.Mention(), banner.Username, entry.Reason))
}

// const weebMessageID = `552788256333234176`
const weebMessageID = `552791672854151190`
const reactionBye = "👋"

func onMessageReactionAdded(s *discordgo.Session, m *discordgo.MessageReactionAdd) {
	targetChannel := serverconfig.Get(m.GuildID, "channel:weeb-channel")
	if targetChannel == "" {
		fmt.Println("No channel set up for weeb channel (good)")
		return
	}

	if m.MessageID == weebMessageID {
		if m.Emoji.Name == reactionBye {
			c, err := s.State.Channel(targetChannel)
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

			err = s.ChannelPermissionSet(targetChannel, m.UserID, "member", 0, discordgo.PermissionReadMessages)
			if err != nil {
				fmt.Println("uh oh something went wrong")
				return
			}

			// s.ChannelMessageSend(m.ChannelID, "added permission")
		}
	}
}

func onMessageReactionRemoved(s *discordgo.Session, m *discordgo.MessageReactionRemove) {
	targetChannel := serverconfig.Get(m.GuildID, "channel:weeb-channel")
	if targetChannel == "" {
		fmt.Println("No channel set up for weeb channel (good)")
		return
	}

	if m.MessageID == weebMessageID {
		if m.Emoji.Name == reactionBye {
			c, err := s.State.Channel(targetChannel)
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

			err = s.ChannelPermissionDelete(targetChannel, m.UserID)
			if err != nil {
				fmt.Println("uh oh something went wrong")
				return
			}
			// s.ChannelMessageSend(m.ChannelID, "removed permission")
		}
	}
}

func onPresenceUpdate(s *discordgo.Session, m *discordgo.PresenceUpdate) {
	return

	fmt.Println("Presence update:", *m)
	fmt.Println("Presence update:", m.Roles)
	if m.User != nil {
		fmt.Println("User", *m.User)
	}
	fmt.Println("Status", m.Status)
	if m.Game != nil {
		fmt.Println("Game", m.Game)
	}
	fmt.Println("Nick", m.Nick)

	user, err := s.User(m.User.ID)
	if err != nil {
		fmt.Println("Error getting user:", err)
		return
	}

	avatarURL := user.AvatarURL("")
	fmt.Println("Avatar URL:", avatarURL)

	filename := path.Base(avatarURL)

	if _, err := os.Stat(filename); err == nil {
		// already exists
		return
	} else if os.IsNotExist(err) {
		resp, err := http.Get(avatarURL)
		if err != nil {
			fmt.Println("Error getting avatar at url", avatarURL)
			return
		}
		defer resp.Body.Close()

		f, err := os.Create(filename)
		if err != nil {
			fmt.Println("Error opening avatar file locally", filename)
			return
		}
		defer f.Close()
		_, err = io.Copy(f, resp.Body)
		if err != nil {
			fmt.Println("Error copying data from request to file")
			return
		}
	}
}

func onMemberJoin(s *discordgo.Session, m *discordgo.GuildMemberAdd, sqlClient *sql.DB) {
	fmt.Println("Member joined:", *m)

	muted, err := mute.IsUserMuted(sqlClient, m.User.ID)
	if err != nil {
		fmt.Println("Error checking user mute:", err)
	} else {
		if muted {
			// Apply muted role
			err = s.GuildMemberRoleAdd(m.GuildID, m.User.ID, config.MutedRole)
			if err != nil {
				fmt.Println("Error assigning role:", err)
			}
		}
	}
}
