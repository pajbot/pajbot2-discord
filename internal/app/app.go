package app

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/pajbot2-discord/internal/api"
	"github.com/pajbot/pajbot2-discord/internal/appctx"
	"github.com/pajbot/pajbot2-discord/internal/channels"
	"github.com/pajbot/pajbot2-discord/internal/config"
	"github.com/pajbot/pajbot2-discord/internal/connection"
	"github.com/pajbot/pajbot2-discord/internal/db"
	"github.com/pajbot/pajbot2-discord/internal/filter"
	"github.com/pajbot/pajbot2-discord/internal/format"
	"github.com/pajbot/pajbot2-discord/internal/roles"
	"github.com/pajbot/pajbot2-discord/internal/serverconfig"
	"github.com/pajbot/pajbot2-discord/pkg"
	"github.com/pajbot/pajbot2-discord/pkg/commands"
	"github.com/pajbot/pajbot2-discord/pkg/utils"

	_ "github.com/lib/pq"

	_ "github.com/pajbot/pajbot2-discord/internal/commands/accountage"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/ban"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/channelinfo"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/channels"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/choice"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/clear"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/color"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/colors"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/configure"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/dev"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/eightball"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/guildinfo"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/modcommands"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/mute"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/ping"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/points"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/profile"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/roleinfo"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/roles"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/roll"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/tags"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/test"
	_ "github.com/pajbot/pajbot2-discord/internal/commands/userid"
)

const (
	timeToKeepLocalAttachments    = 10 * time.Minute
	timeBetweenAttachmentCleanups = 15 * time.Minute
)

type localAttachment struct {
	reader   io.Reader
	filename string
}

var attachments = map[string][]*discordgo.MessageAttachment{}
var attachmentsMutex = sync.Mutex{}

const forsenServerID = `97034666673975296`

type App struct {
	bot *discordgo.Session

	FilterRunner *filter.Runner

	sqlClient *sql.DB
}

func New(bot *discordgo.Session, sqlClient *sql.DB) *App {
	return &App{
		bot:          bot,
		FilterRunner: filter.NewRunner(),

		sqlClient: sqlClient,
	}
}

func fetchThreadStarterMessage(s *discordgo.Session, thread *discordgo.Channel) (*discordgo.Message, error) {
	channelIDs := []string{thread.ID}
	if thread.ParentID != "" {
		channelIDs = append(channelIDs, thread.ParentID)
	}

	var err error
	for _, channelID := range channelIDs {
		var message *discordgo.Message
		message, err = s.ChannelMessage(channelID, thread.ID)
		if err == nil {
			return message, nil
		}
	}

	return nil, err
}

func storeMessageLocally(m *discordgo.Message) error {
	if err := storeMessageAttachmentsLocally(m); err != nil {
		return err
	}

	if err := db.PushMessageIntoDatabase(m); err != nil {
		return fmt.Errorf("error pushing message into database: %w", err)
	}

	return nil
}

func storeMessageAttachmentsLocally(m *discordgo.Message) error {
	for _, a := range m.Attachments {
		attachmentsMutex.Lock()
		attachments[m.ID] = append(attachments[m.ID], a)
		attachmentsMutex.Unlock()
		attachmentLocalPath := filepath.Join("attachments", fmt.Sprintf("%s-%s", a.ID, a.Filename))

		if _, err := os.Stat(attachmentLocalPath); os.IsNotExist(err) {
			resp, err := http.Get(a.URL)
			if err != nil {
				return fmt.Errorf("error getting attachment at url %s: %w", a.URL, err)
			}
			defer resp.Body.Close()

			f, err := os.Create(attachmentLocalPath)
			if err != nil {
				return fmt.Errorf("error opening attachment file locally %s: %w", attachmentLocalPath, err)
			}
			defer f.Close()
			if _, err = io.Copy(f, resp.Body); err != nil {
				return fmt.Errorf("error copying data from request to file: %w", err)
			}
		}
	}

	return nil
}

func storeThreadStarterMessageLocally(m *discordgo.Message, threadTitle string) error {
	if err := storeMessageAttachmentsLocally(m); err != nil {
		return err
	}

	if err := db.PushThreadMessageIntoDatabase(m, threadTitle); err != nil {
		return fmt.Errorf("error pushing thread starter message into database: %w", err)
	}

	return nil
}

func (a *App) RegisterUser(pendingConnection *connection.PendingConnection, twitchUser connection.TwitchValidateResponse) {
	defer connection.UnregisterPendingConnection(pendingConnection.ConnectionID)
	membersRole := roles.GetSingle(pendingConnection.DiscordGuildID, "member")
	if membersRole == "" {
		fmt.Println("Error applying members role: There's no members role in the server ??")
		return
	}

	if err := a.bot.GuildMemberRoleAdd(pendingConnection.DiscordGuildID, pendingConnection.DiscordUserID, membersRole); err != nil {
		fmt.Println("Error adding members role to user:", pendingConnection, twitchUser, err)
	}

	existingConnection, err := connection.Get(a.sqlClient, pendingConnection.DiscordGuildID, pendingConnection.DiscordUserID)
	if err != nil {
		fmt.Println("error getting existing connection:", pendingConnection, twitchUser, err)
		return
	}

	member, err := a.bot.GuildMember(pendingConnection.DiscordGuildID, pendingConnection.DiscordUserID)
	if err != nil {
		fmt.Println("error getting member:", pendingConnection, twitchUser, err)
		return
	}

	if err := connection.Upsert(a.sqlClient, pendingConnection.DiscordGuildID, pendingConnection.DiscordUserID, member.User.Username, twitchUser.UserID, twitchUser.Login); err != nil {
		fmt.Println("Error upserting into db:", pendingConnection, twitchUser, err)
		return
	}

	a.postConnectionUpdate(member, existingConnection, twitchUser)

	if err := roles.Grant(a.bot, pendingConnection.DiscordGuildID, pendingConnection.DiscordUserID, "member"); err != nil {
		fmt.Println("Error adding Members role to user:", err)
	} else {
		fmt.Printf("Granted Members role to %s (%s)\n", member.User.Username, member.User.ID)
	}
}

func (a *App) postConnectionUpdate(member *discordgo.Member, existingConnection *connection.DiscordConnection, twitchUser connection.TwitchValidateResponse) {
	embed := &discordgo.MessageEmbed{
		Title: "User Connection Update",
	}
	targetChannel := channels.Get(member.GuildID, "connection-updates", "system-messages")
	if targetChannel == "" {
		fmt.Println("No channel set up for connection updates")
		return
	}

	embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
		URL:    member.User.AvatarURL("64x64"),
		Width:  64,
		Height: 64,
	}

	payload := utils.MentionMember(member)
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Name", Value: payload})

	fmt.Println(existingConnection)

	if existingConnection != nil {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Previous connection",
			Value:  format.Connection(existingConnection),
			Inline: false,
		})
	}

	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "New connection",
		Value:  fmt.Sprintf("`%s` (TUID `%s`)", utils.EscapeCodeBlock(twitchUser.Login), twitchUser.UserID),
		Inline: false,
	})

	altConnections, err := connection.GetByTwitchID(a.sqlClient, twitchUser.UserID, member.GuildID, member.User.ID)
	if err != nil {
		fmt.Println("Error getting alt connections:", err)
	} else {
		if len(altConnections) > 0 {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "Alt Detection",
				Value:  format.AltConnections(altConnections),
				Inline: false,
			})
		}
	}

	if _, err := a.bot.ChannelMessageSendEmbed(targetChannel, embed); err != nil {
		fmt.Println("Error sending message embed:", err)
	}
}

func (a *App) StartWebServer(ctx context.Context) error {

	wrapApp := func(a *App, next *http.ServeMux) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c := context.WithValue(r.Context(), appctx.Application, a)

			next.ServeHTTP(w, r.WithContext(c))
		})
	}

	go func() {
		api.Init()

		handler := http.NewServeMux()

		api.Attach(handler)

		realHandler := wrapApp(a, handler)

		s := &http.Server{
			Addr:    config.WebListenAddr,
			Handler: realHandler,
		}

		if err := s.ListenAndServe(); err != nil {
			fmt.Println("Error in http listen and serve:", err)
		}
	}()

	return nil
}

func (a *App) PostToActionLog(guildID, sourceChannelID, title string, fields []*discordgo.MessageEmbedField) {
	if serverconfig.ActionLogChannelIgnored(guildID, sourceChannelID) {
		return
	}

	targetChannel := channels.Get(guildID, "action-log")
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

func (a *App) OnMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from bots
	if m.Message.Author.Bot {
		return
	}

	// Remove nitro colors if the user doesn't have nitro
	if hasAccess, _ := utils.MemberInRoles(s, m.GuildID, m.Author.ID, "nitrobooster"); !hasAccess {
		colorRoles := utils.ColorRoles(s, m.GuildID)
		messageTimestamp, messageTimestampErr := discordgo.SnowflakeTimestamp(m.Message.ID)
		anyRemoved, err := utils.RemoveNitroColors(s, m.GuildID, m.Author.ID, colorRoles)
		if err != nil {
			fmt.Println("Error Removing nitro from", m.Author.ID, messageTimestamp, messageTimestampErr, err)
		} else {
			if anyRemoved {
				fmt.Println("Removing nitro from", m.Author.ID, messageTimestamp, messageTimestampErr)
			}
		}
	}

	if m.GuildID == forsenServerID {
		if strings.Contains(m.Message.Content, "bajscran") {
			if err := s.MessageReactionAdd(m.ChannelID, m.ID, "⬅️"); err != nil {
				fmt.Println("error adding left:", err)
			}
			if err := s.MessageReactionAdd(m.ChannelID, m.ID, "➡️"); err != nil {
				fmt.Println("error adding right:", err)
			}
		}
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
				Value:  strings.ReplaceAll(m.Content, "`", ""),
				Inline: true,
			})
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "Channel",
				Value:  "<#" + m.ChannelID + ">",
				Inline: true,
			})

			targetChannel := channels.Get(m.GuildID, "action-log")
			if serverconfig.ActionLogChannelIgnored(m.GuildID, m.ChannelID) {
				return
			}
			if targetChannel == "" {
				fmt.Println("No channel set up for moderation actions")
				return
			}

			// Announce mute in moderation-action channel
			s.ChannelMessageSendEmbed(targetChannel, embed)
			return
		}
	}

	// Remove messages containing "display links"
	linkRegex := regexp.MustCompile(`\[(.*?)\]\((https?://\S+)\)`)
	matches := linkRegex.FindAllStringSubmatch(m.Message.Content, -1)
	if len(matches) > 0 {
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
				Title: "Message deleted because it contained a link with a different display text",
			}
			payload := fmt.Sprintf("<@%s> - Name: %s#%s - ID: %s", m.Author.ID, m.Author.Username, m.Author.Discriminator, m.Author.ID)
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "Author",
				Value:  payload,
				Inline: true,
			})
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "Content",
				Value:  strings.ReplaceAll(m.Content, "`", ""),
				Inline: true,
			})
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "Channel",
				Value:  "<#" + m.ChannelID + ">",
				Inline: true,
			})

			targetChannel := channels.Get(m.GuildID, "action-log")
			if serverconfig.ActionLogChannelIgnored(m.GuildID, m.ChannelID) {
				return
			}
			if targetChannel == "" {
				fmt.Println("No channel set up for moderation actions")
				return
			}

			s.ChannelMessageSendEmbed(targetChannel, embed)
			return
		}
	}

	if err := storeMessageLocally(m.Message); err != nil {
		log.Println(err)
	}

	if m.Author.ID == s.State.User.ID {
		return
	}

	autoReactIDs := serverconfig.GetAutoReact(m.GuildID, m.ChannelID)
	for _, emojiID := range autoReactIDs {
		err := s.MessageReactionAdd(m.ChannelID, m.ID, emojiID)
		if err != nil {
			fmt.Println("ERR REACT:", err)
		}
	}

	a.FilterRunner.ScanMessage(filter.Message{
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

func (a *App) OnMessageEdited(s *discordgo.Session, m *discordgo.MessageUpdate) {
	messageContent, authorID, err := db.GetMessageFromDatabase(m.ID)
	if err != nil {
		fmt.Printf("on message edit: Error getting full message '%s': %s\n", m.ID, err)
		return
	}
	if !serverconfig.ActionLogChannelIgnored(m.GuildID, m.ChannelID) {
		targetChannel := channels.Get(m.GuildID, "action-log")
		if targetChannel == "" {
			fmt.Println("No channel set up for action log")
		} else {
			// Try to get member
			var member *discordgo.Member
			if authorID != "unknown" {
				member, err = s.GuildMember(m.GuildID, authorID)
				if err != nil {
					fmt.Println("Error getting guild member of edited message:", err)
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
					Value:  strings.ReplaceAll(messageContent, "`", ""),
					Inline: true,
				})
			}
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "New message",
				Value:  strings.ReplaceAll(m.Content, "`", ""),
				Inline: true,
			})
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "Channel",
				Value:  "<#" + m.ChannelID + ">",
				Inline: true,
			})
			idTime, idTimeErr := discordgo.SnowflakeTimestamp(m.ID)
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "Snowflake/ID",
				Value:  fmt.Sprintf("%s: %s %s", m.ID, idTime, idTimeErr),
				Inline: true,
			})
			s.ChannelMessageSendEmbed(targetChannel, embed)
		}
	}

	err = db.PushMessageIntoDatabase(m.Message)
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
				Value:  strings.ReplaceAll(m.Content, "`", ""),
				Inline: true,
			})
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "Channel",
				Value:  "<#" + m.ChannelID + ">",
				Inline: true,
			})

			targetChannel := channels.Get(m.GuildID, "action-log")
			if serverconfig.ActionLogChannelIgnored(m.GuildID, m.ChannelID) {
				return
			}
			if targetChannel == "" {
				fmt.Println("No channel set up for moderation actions")
				return
			}

			// Announce mute in moderation-action channel
			s.ChannelMessageSendEmbed(targetChannel, embed)
			return
		}
	}
	a.FilterRunner.ScanMessage(filter.Message{
		DiscordMessage: m.Message,
	})
}

// TODO: Don't list messages deleted from or by ourselves (this bot account)
func (a *App) OnMessageDeleted(s *discordgo.Session, m *discordgo.MessageDelete) {
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
	messageContent, authorID, err := db.GetMessageFromDatabase(m.ID)
	if err != nil {
		fmt.Println("on message deleted: Error getting full message")
		if m.BeforeDelete != nil {
			messageContent = m.BeforeDelete.Content
			authorID = m.BeforeDelete.Author.ID
		}
	}
	if serverconfig.ActionLogChannelIgnored(m.GuildID, m.ChannelID) {
		return
	}

	targetChannel := channels.Get(m.GuildID, "action-log")
	if targetChannel == "" {
		fmt.Println("No channel set up for action log")
		return
	}

	if targetChannel == m.ChannelID {
		// Skipping messages deleted from the action log channel itself
		return
	}

	messageAge := time.Since(creationTime)
	if messageAge > 300*24*time.Hour {
		fmt.Printf("Skipping message deletion log of %s since it's too old (%s)\n", m.ID, messageAge)
		return
	}

	// Try to get member
	var member *discordgo.Member
	if authorID != "unknown" {
		member, err = s.GuildMember(m.GuildID, authorID)
		if err != nil {
			fmt.Printf("Error getting guild member (%s): %s\n", authorID, err)
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
			Value:  fmt.Sprintf("unknown (<@%s>)", authorID),
			Inline: true,
		})
	}
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "Content",
		Value:  strings.ReplaceAll(messageContent, "`", ""),
		Inline: true,
	})
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "Channel",
		Value:  "<#" + m.ChannelID + ">",
		Inline: true,
	})
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "Message ID",
		Value:  m.ID,
		Inline: true,
	})
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "Message creation time",
		Value:  creationTime.Format("2006-01-02 15:04:05"),
		Inline: true,
	})
	s.ChannelMessageSendEmbed(targetChannel, embed)
	for i, file := range attachmentsToSend {
		content := fmt.Sprintf("Attachment %d/%d for message %s\nPosted by ID: %s", i+1, len(attachmentsToSend), m.ID, authorID)
		if member != nil {
			content = fmt.Sprintf("Attachment %d/%d for message %s\nPosted by Name: %s#%s - ID: %s", i+1, len(attachmentsToSend), m.ID, member.User.Username, member.User.Discriminator, authorID)
		}
		s.ChannelFileSendWithMessage(targetChannel, content, file.filename, file.reader)
	}
}

func onThreadStart(s *discordgo.Session, thread *discordgo.Channel) {
	if thread == nil || !thread.IsThread() {
		return
	}

	switch thread.Type {
	case discordgo.ChannelTypeGuildPublicThread, discordgo.ChannelTypeGuildNewsThread:
	default:
		return
	}

	message, err := fetchThreadStarterMessage(s, thread)
	if err != nil {
		fmt.Printf("on thread start: Error getting starter message '%s': %s\n", thread.ID, err)
		return
	}

	if err := storeThreadStarterMessageLocally(message, thread.Name); err != nil {
		log.Println("Error storing thread starter message:", err)
	}
}

func (a *App) OnThreadCreated(s *discordgo.Session, m *discordgo.ThreadCreate) {
	onThreadStart(s, m.Channel)
}

func (a *App) OnThreadUpdated(s *discordgo.Session, m *discordgo.ThreadUpdate) {
	onThreadStart(s, m.Channel)
}
