package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/pajbot2-discord/internal/autoreact"
	"github.com/pajbot/pajbot2-discord/internal/config"
	"github.com/pajbot/pajbot2-discord/internal/filter"
	"github.com/pajbot/pajbot2-discord/internal/mute"
	"github.com/pajbot/pajbot2-discord/internal/serverconfig"
	"github.com/pajbot/pajbot2-discord/pkg"
	"github.com/pajbot/pajbot2-discord/pkg/commands"
	"github.com/pajbot/pajbot2-discord/pkg/utils"
	sharedutils "github.com/pajbot/utils"
	"github.com/pajlada/stupidmigration"

	_ "github.com/lib/pq"

	_ "github.com/pajbot/pajbot2-discord/internal/commands/accountage"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/ban"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/channelinfo"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/channels"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/clear"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/color"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/colors"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/configure"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/eightball"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/guildinfo"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/modcommands"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/mute"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/ping"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/choice"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/points"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/profile"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/roleinfo"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/roles"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/tags"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/test"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/userid"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/whereisstreamer"
)

const (
	timeToKeepLocalAttachments    = 10 * time.Minute
	timeBetweenAttachmentCleanups = 15 * time.Minute
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

	serverconfig.Load(sqlClient)
}

type App struct {
	bot *discordgo.Session

	filterRunner *filter.Runner
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bot, err := discordgo.New("Bot " + config.Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	bot.Identify.Intents = discordgo.IntentsAllWithoutPrivileged | discordgo.IntentsGuildMembers

	bot.State.MaxMessageCount = 1000

	var app App

	app.bot = bot
	app.filterRunner = filter.NewRunner()

	// app.filterRunner.Dry = true

	app.filterRunner.OnDelete(func(message filter.Message) {
		fmt.Println("delete message...")
		if err := bot.ChannelMessageDelete(message.DiscordMessage.ChannelID, message.DiscordMessage.ID); err != nil {
			fmt.Println("Error deleting message:", err)
		}

		// TODO: include which action it matched
		app.postToActionLog(message.DiscordMessage.GuildID, "Deleting message because it matched a filter",
			[]*discordgo.MessageEmbedField{
				{
					Name:   "Message",
					Value:  message.DiscordMessage.Content,
					Inline: false,
				},
			})
	})
	app.filterRunner.OnMute(func(message filter.Message) {
		fmt.Println("mute user...")
	})
	app.filterRunner.OnBan(func(message filter.Message) {
		fmt.Println("ban user...")
	})

	// app.filterRunner.OnDelete(...)
	// app.filterRunner.OnBan(...)
	// app.filterRunner.OnMute(...)

	bot.AddHandler(app.onMessage)
	bot.AddHandler(onMessageDeleted)
	bot.AddHandler(app.onMessageEdited)
	bot.AddHandler(onUserBanned)
	bot.AddHandler(onUserJoined)
	bot.AddHandler(onUserLeft)
	bot.AddHandler(onMessageReactionAdded)
	bot.AddHandler(onMessageReactionRemoved)
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

	// Run queued up actions (e.g. unmute user)
	go startActionRunner(ctx, bot)
	// Run queued up unmutes
	go startUnmuterRunner(ctx, bot)
	// Clean up attachments occasionally
	go startCleanUpAttachments(ctx)

	go app.filterRunner.Run(ctx)

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

func pushMessageIntoDatabase(m *discordgo.Message) (err error) {
	const query = `INSERT INTO discord_messages (id, content, author_id) VALUES ($1, $2, $3) ON CONFLICT (id) DO UPDATE SET content=$2`
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

var attachments = map[string][]*discordgo.MessageAttachment{}
var attachmentsMutex = sync.Mutex{}

const noInvites = true

func (a *App) postToActionLog(guildID, title string, fields []*discordgo.MessageEmbedField) {
	targetChannel := serverconfig.Get(guildID, "channel:action-log")
	if targetChannel == "" {
		fmt.Println("No channel set up for action log")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:  title,
		Fields: fields,
	}

	a.bot.ChannelMessageSendEmbed(targetChannel, embed)
}

func (a *App) onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from bots
	if m.Message.Author.Bot {
		return
	}

	// Remove nitro colors if the user doesn't have nitro
	if hasAccess, _ := utils.MemberInRoles(s, m.GuildID, m.Author.ID, "nitrobooster"); !hasAccess {
		colorRoles := utils.ColorRoles(s, m.GuildID)
		utils.RemoveNitroColors(s, m.GuildID, m.Author.ID, colorRoles)
	}

	if inviteCode, ok := utils.ResolveInviteCode(m.Message.Content); ok && inviteCode != "forsen" {
		hasAccess, err := utils.MemberInRoles(s, m.GuildID, m.Author.ID, "minimod")
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		if !hasAccess {
			err := s.ChannelMessageDelete(m.Message.ChannelID, m.Message.ID)
			if err != nil {
				fmt.Println("Error deleting message")
			}
			embed := &discordgo.MessageEmbed{
				Title: "Message deleted because it contained a server invite",
			}
			payload := fmt.Sprintf("<@%s> - Name: %s#%s - ID: %s", m.Author.ID, m.Author.Username, m.Author.Discriminator, m.Author.ID)
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "Author",
				Value:  payload,
				Inline: true,
			})
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "Content",
				Value:  strings.Replace(m.Content, "`", "", -1),
				Inline: true,
			})
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "Channel",
				Value:  "<#" + m.ChannelID + ">",
				Inline: true,
			})

			targetChannel := serverconfig.Get(m.GuildID, "channel:action-log")
			if targetChannel == "" {
				fmt.Println("No channel set up for moderation actions")
				return
			}

			// Announce mute in moderation-action channel
			s.ChannelMessageSendEmbed(targetChannel, embed)
			return
		}
	}

	for _, a := range m.Message.Attachments {
		attachmentsMutex.Lock()
		attachments[m.Message.ID] = append(attachments[m.Message.ID], a)
		attachmentsMutex.Unlock()
		attachmentLocalPath := filepath.Join("attachments", fmt.Sprintf("%s-%s", a.ID, a.Filename))

		if _, err := os.Stat(attachmentLocalPath); os.IsNotExist(err) {
			resp, err := http.Get(a.URL)
			if err != nil {
				fmt.Println("Error getting avatar at url", a.URL)
				return
			}
			defer resp.Body.Close()

			f, err := os.Create(attachmentLocalPath)
			if err != nil {
				fmt.Println("Error opening avatar file locally", attachmentLocalPath)
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

	// push message into database
	err := pushMessageIntoDatabase(m.Message)
	if err != nil {
		log.Println("Error pushing message into databasE:", err)
	}

	if m.Author.ID == s.State.User.ID {
		return
	}

	autoReactIDs := autoreact.Get(m.GuildID, m.ChannelID)
	for _, emojiID := range autoReactIDs {
		err := s.MessageReactionAdd(m.ChannelID, m.ID, emojiID)
		if err != nil {
			fmt.Println("ERR REACT:", err)
		}
	}

	a.filterRunner.ScanMessage(filter.Message{
		DiscordMessage: m.Message,
	})

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

func (a *App) onMessageEdited(s *discordgo.Session, m *discordgo.MessageUpdate) {
	messageContent, authorID, err := getMessageFromDatabase(m.ID)
	if err != nil {
		fmt.Println("on message edit: Error getting full message")
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
		Title: "Message edited",
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
	if messageContent != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Old message",
			Value:  strings.Replace(messageContent, "`", "", -1),
			Inline: true,
		})
	}
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "New message",
		Value:  strings.Replace(m.Content, "`", "", -1),
		Inline: true,
	})
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "Channel",
		Value:  "<#" + m.ChannelID + ">",
		Inline: true,
	})
	s.ChannelMessageSendEmbed(targetChannel, embed)

	err = pushMessageIntoDatabase(m.Message)
	if err != nil {
		log.Println("Error pushing message into databasE (from edit):", err)
	}

	if inviteCode, ok := utils.ResolveInviteCode(m.Message.Content); ok && inviteCode != "forsen" {
		hasAccess, err := utils.MemberInRoles(s, m.GuildID, m.Author.ID, "minimod")
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		if !hasAccess {
			err := s.ChannelMessageDelete(m.Message.ChannelID, m.Message.ID)
			if err != nil {
				fmt.Println("Error deleting message")
			}
			embed := &discordgo.MessageEmbed{
				Title: "Message deleted because it contained a server invite",
			}
			payload := fmt.Sprintf("<@%s> - Name: %s#%s - ID: %s", m.Author.ID, m.Author.Username, m.Author.Discriminator, m.Author.ID)
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "Author",
				Value:  payload,
				Inline: true,
			})
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "Content",
				Value:  strings.Replace(m.Content, "`", "", -1),
				Inline: true,
			})
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "Channel",
				Value:  "<#" + m.ChannelID + ">",
				Inline: true,
			})

			targetChannel := serverconfig.Get(m.GuildID, "channel:action-log")
			if targetChannel == "" {
				fmt.Println("No channel set up for moderation actions")
				return
			}

			// Announce mute in moderation-action channel
			s.ChannelMessageSendEmbed(targetChannel, embed)
			return
		}
	}
	a.filterRunner.ScanMessage(filter.Message{
		DiscordMessage: m.Message,
	})
}

type localAttachment struct {
	reader   io.Reader
	filename string
}

// TODO: Don't list messages deleted from or by ourselves (this bot account)
func onMessageDeleted(s *discordgo.Session, m *discordgo.MessageDelete) {
	creationTime, err := utils.CreationTime(m.ID)
	if err != nil {
		panic(err)
	}
	var attachmentsToSend []localAttachment
	if time.Now().Before(creationTime.Add(timeToKeepLocalAttachments)) {
		if messageAttachments, ok := attachments[m.Message.ID]; ok {
			for _, a := range messageAttachments {
				attachmentLocalPath := filepath.Join("attachments", fmt.Sprintf("%s-%s", a.ID, a.Filename))
				file, err := os.Open(attachmentLocalPath)
				if err != nil {
					fmt.Println("Error opening file", err)
					continue
				}

				attachmentsToSend = append(attachmentsToSend, localAttachment{
					reader:   file,
					filename: a.Filename,
				})

				// TODO: Wait with posting the message deleted message until we're done downloading all attachments
			}
		}
	}
	var authorID string
	messageContent, authorID, err := getMessageFromDatabase(m.ID)
	if err != nil {
		fmt.Println("on message deleted: Error getting full message")
		if m.BeforeDelete != nil {
			messageContent = m.BeforeDelete.Content
			authorID = m.BeforeDelete.Author.ID
		}
	}

	targetChannel := serverconfig.Get(m.GuildID, "channel:action-log")
	if targetChannel == "" {
		fmt.Println("No channel set up for action log")
		return
	}

	if targetChannel == m.ChannelID {
		// Skipping messages deleted from the action log channel itself
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
	for i, file := range attachmentsToSend {
		content := fmt.Sprintf("Attachment %d/%d for message %s\nPosted by Name: %s#%s - ID: %s", i+1, len(attachmentsToSend), m.ID, member.User.Username, member.User.Discriminator, authorID)
		s.ChannelFileSendWithMessage(targetChannel, content, file.filename, file.reader)
	}
}

func onUserBanned(s *discordgo.Session, m *discordgo.GuildBanAdd) {
	go func() {
		time.Sleep(5 * time.Second)

		auditLog, err := s.GuildAuditLog(m.GuildID, "", "", int(discordgo.AuditLogActionMemberBanAdd), 50)
		if err != nil {
			fmt.Println("Error getting user ban data", err)
			return
		}
		for _, entry := range auditLog.AuditLogEntries {
			if entry.TargetID != m.User.ID {
				continue
			}

			// Found ban for user, hope this is the right one xD

			banner, err := s.User(entry.UserID)
			if err != nil {
				fmt.Println("Error getting member state for banner:", err)
				return
			}
			botUser, err := s.User("@me")
			if err == nil && banner.ID == botUser.ID {
				fmt.Println("Ban is initiated by the bot, will not log into moderator actions channel")
				return
			}

			targetChannel := serverconfig.Get(m.GuildID, "channel:moderation-action")
			if targetChannel == "" {
				fmt.Println("No channel set up for moderation actions")
				return
			}

			hidden := strings.Contains(entry.Reason, "!hide") || strings.Contains(entry.Reason, "!hidden")

			anon := strings.Contains(entry.Reason, "!anon")

			var message string

			if anon {
				message = fmt.Sprintf("%s was banned for reason: %s", utils.MentionUser(s, m.GuildID, m.User), utils.EscapeMarkdown(entry.Reason))
			} else {
				message = fmt.Sprintf("%s banned %s for reason: %s", utils.MentionUserFromParts(s, m.GuildID, banner.ID, banner.Username, banner.Discriminator), utils.MentionUser(s, m.GuildID, m.User), utils.EscapeMarkdown(entry.Reason))
			}

			if hidden {
				fmt.Println("HIDDEN BAN:", message)
			} else {
				s.ChannelMessageSend(targetChannel, message)
			}

			return
		}

		fmt.Println("Unable to find log for banned user Pepega")
	}()
}

func postUserInfo(s *discordgo.Session, member *discordgo.Member, title string) {
	embed := &discordgo.MessageEmbed{
		Title: title,
	}
	targetChannel := serverconfig.Get(member.GuildID, "channel:system-messages")
	if targetChannel == "" {
		fmt.Println("No channel set up for system messages")
		return
	}

	accountCreationDate, err := discordgo.SnowflakeTimestamp(member.User.ID)
	if err != nil {
		fmt.Println("error getting user created date:", err)
		return
	}

	embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
		URL:    member.User.AvatarURL("64x64"),
		Width:  64,
		Height: 64,
	}

	payload := utils.MentionMember(member)
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Name", Value: payload})
	payload = fmt.Sprintf("%s (%s ago)", accountCreationDate.Format("2006-01-02 15:04:05"), sharedutils.TimeSince(accountCreationDate))
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Account Created", Value: payload})

	_, err = s.ChannelMessageSendEmbed(targetChannel, embed)
	if err != nil {
		fmt.Println("Error sending message embed:", err)
	}
}

func banIfUserIsYoungerThan(s *discordgo.Session, m *discordgo.GuildMemberAdd, minimumAge time.Duration) {
	accountCreationDate, err := discordgo.SnowflakeTimestamp(m.Member.User.ID)
	if err != nil {
		fmt.Println("error getting user created date:", err)
		return
	}

	accountAge := time.Since(accountCreationDate)

	if accountAge < minimumAge {
		delay := 5 + rand.Intn(26)
		fmt.Printf("Banning user with id %s in %d seconds\n", m.Member.User.ID, delay)

		go func() {
			time.Sleep(time.Duration(delay) * time.Second)
			err := s.GuildBanCreateWithReason(m.GuildID, m.Member.User.ID, "!hide", 1)
			if err != nil {
				fmt.Println("Error banning user:", err)
			}
		}()
	}
}

func onUserJoined(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	postUserInfo(s, m.Member, "User Joined")

	// banIfUserIsYoungerThan(s, m, 1*time.Hour)
}

func onUserLeft(s *discordgo.Session, m *discordgo.GuildMemberRemove) {
	postUserInfo(s, m.Member, "User Left")
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
			var overwriteDenies int64
			for _, overwrite := range c.PermissionOverwrites {
				if overwrite.Type == discordgo.PermissionOverwriteTypeMember && overwrite.ID == m.UserID {
					overwriteDenies = overwrite.Deny
				}
			}
			if overwriteDenies != 0 {
				// s.ChannelMessageSend(m.ChannelID, "cannot set your permissions - you have weird permissions set from before")
				return
			}

			err = s.ChannelPermissionSet(targetChannel, m.UserID, discordgo.PermissionOverwriteTypeMember, 0, discordgo.PermissionViewChannel)
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
			var overwriteDenies int64
			for _, overwrite := range c.PermissionOverwrites {
				if overwrite.Type == discordgo.PermissionOverwriteTypeMember && overwrite.ID == m.UserID {
					overwriteDenies = overwrite.Deny
				}
			}

			if overwriteDenies != discordgo.PermissionViewChannel {
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

func onMemberJoin(s *discordgo.Session, m *discordgo.GuildMemberAdd, sqlClient *sql.DB) {
	err := mute.ReapplyMute(s, sqlClient, m)
	if err != nil {
		fmt.Println("Error when seeing if we need to reapply mute:", err)
	}
}
