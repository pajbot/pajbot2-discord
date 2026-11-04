package main

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/pajbot2-discord/internal/app"
	"github.com/pajbot/pajbot2-discord/internal/channels"
	"github.com/pajbot/pajbot2-discord/internal/config"
	"github.com/pajbot/pajbot2-discord/internal/connection"
	"github.com/pajbot/pajbot2-discord/internal/db"
	"github.com/pajbot/pajbot2-discord/internal/filter"
	"github.com/pajbot/pajbot2-discord/internal/mute"
	"github.com/pajbot/pajbot2-discord/internal/roles"
	"github.com/pajbot/pajbot2-discord/internal/serverconfig"
	"github.com/pajbot/pajbot2-discord/internal/slashcommands"
	"github.com/pajbot/pajbot2-discord/internal/twitchstreamannouncer"
	"github.com/pajbot/pajbot2-discord/internal/values"
	"github.com/pajbot/pajbot2-discord/pkg/commands"
	"github.com/pajbot/pajbot2-discord/pkg/utils"
	sharedutils "github.com/pajbot/utils"
	normalize "github.com/pajlada/lidl-normalize"

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

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	commands.SQLClient = db.SQLClient

	serverconfig.Load(db.SQLClient)

	slashcommands.Initialize(db.SQLClient)

	<-rdyHelixClient
	fmt.Println("Twitch API initialized")

	bot, err := discordgo.New("Bot " + config.Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	bot.Identify.Intents = discordgo.IntentsAllWithoutPrivileged | discordgo.IntentsGuildMembers

	bot.State.MaxMessageCount = 1000

	app := app.New(bot, db.SQLClient)

	app.FilterRunner.AddFilter(&filter.Delete{
		Checker: func(content string) bool {
			requiredParts := []string{"larvalabs.net", "cryptopunks", "nft", "tokens"}

			postNormalize, err := normalize.Normalize(content)
			if err != nil {
				fmt.Println("Error normalizing message:", content)
				return false
			}

			postNormalize = strings.ToLower(postNormalize)

			for _, p := range requiredParts {
				if !strings.Contains(postNormalize, p) {
					return false
				}
			}

			fmt.Println("The message", content, "post normalize:", postNormalize, "contained all bad parts. delete message!!")

			return true
		},
	})

	app.FilterRunner.AddFilter(&filter.Delete{
		Checker: func(content string) bool {
			requiredParts := []string{"cryptopunks", "wallet", "free"}

			postNormalize, err := normalize.Normalize(content)
			if err != nil {
				fmt.Println("Error normalizing message:", content)
				return false
			}

			postNormalize = strings.ToLower(postNormalize)

			for _, p := range requiredParts {
				if !strings.Contains(postNormalize, p) {
					return false
				}
			}

			fmt.Println("The message", content, "post normalize:", postNormalize, "contained all bad parts. delete message!!")

			return true
		},
	})

	app.FilterRunner.AddFilter(&filter.Delete{
		Checker: func(content string) bool {
			requiredParts := []string{"ethereum", "airdrop", "claim"}

			postNormalize, err := normalize.Normalize(content)
			if err != nil {
				fmt.Println("Error normalizing message:", content)
				return false
			}

			postNormalize = strings.ToLower(postNormalize)

			for _, p := range requiredParts {
				if !strings.Contains(postNormalize, p) {
					return false
				}
			}

			fmt.Println("The message", content, "post normalize:", postNormalize, "contained all bad parts. delete message!!")

			return true
		},
	})

	app.FilterRunner.AddFilter(&filter.DeleteAdvanced{
		Checker: func(message *discordgo.Message) bool {
			bannedHosts := []string{
				"tenor.com",
			}
			if message.ChannelID != "103642197076742144" {
				return false
			}

			for _, embed := range message.Embeds {
				lowercaseURL := strings.ToLower(embed.URL)
				for _, bannedHost := range bannedHosts {
					if strings.Contains(lowercaseURL, bannedHost) {
						return true
					}
				}
			}

			return false
		},
	})

	app.FilterRunner.OnDelete(func(message filter.Message) {
		fmt.Println("delete message...")
		if err := bot.ChannelMessageDelete(message.DiscordMessage.ChannelID, message.DiscordMessage.ID); err != nil {
			fmt.Println("Error deleting message:", err)
		}

		// TODO: include which action it matched
		app.PostToActionLog(message.DiscordMessage.GuildID, message.DiscordMessage.ChannelID, "Deleting message because it matched a filter",
			[]*discordgo.MessageEmbedField{
				{
					Name:   "Message",
					Value:  message.DiscordMessage.Content,
					Inline: false,
				},
			})
	})
	app.FilterRunner.OnMute(func(message filter.Message) {
		fmt.Println("mute user...")
	})
	app.FilterRunner.OnBan(func(message filter.Message) {
		fmt.Println("ban user...")
	})

	// app.FilterRunner.OnDelete(...)
	// app.FilterRunner.OnBan(...)
	// app.FilterRunner.OnMute(...)

	go twitchstreamannouncer.Start(ctx, helixClient, bot, db.SQLClient)

	bot.AddHandler(app.OnMessage)
	bot.AddHandler(app.OnMessageDeleted)
	bot.AddHandler(app.OnThreadCreated)
	bot.AddHandler(onThreadDeleted)
	bot.AddHandler(app.OnThreadUpdated)
	bot.AddHandler(app.OnMessageEdited)
	bot.AddHandler(onUserBanned)
	bot.AddHandler(onUserLeft)
	bot.AddHandler(onMessageReactionAdded)
	bot.AddHandler(onMessageReactionRemoved)
	bot.AddHandler(func(s *discordgo.Session, m *discordgo.GuildCreate) {
		// Check streamer's live status occasionally and post it in the stream-status channel on updates
		twitchstreamannouncer.Register(m.Guild.ID)
	})
	bot.AddHandler(func(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
		onMemberJoin(s, m, db.SQLClient)
	})

	if err := app.StartWebServer(ctx); err != nil {
		panic(err)
	}

	// Open a websocket connection to Discord and begin listening.
	err = bot.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	defer bot.Close()

	slashcommands := slashcommands.New(config.SlashCommandGuildIDs)

	// Create all slash commands
	err = slashcommands.Create(bot)
	if err != nil {
		fmt.Println("Error creating slash commands:", err)
		return
	}

	// Run queued up actions (e.g. unmute user)
	go startActionRunner(ctx, bot)
	// Run queued up unmutes
	go startUnmuterRunner(ctx, bot)
	// Clean up attachments occasionally
	go startCleanUpAttachments(ctx)

	go app.FilterRunner.Run(ctx)

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	fmt.Println("Exiting")

	// Delete all registered slash commands
	err = slashcommands.Delete(bot)
	if err != nil {
		fmt.Println("Error deleting slash commands:", err)
		return
	}
	fmt.Println("Deleted slash commands")
}

const noInvites = true

func onThreadDeleted(s *discordgo.Session, m *discordgo.ThreadDelete) {
	messageContent, authorID, threadTitle, err := db.GetThreadMessageFromDatabase(m.ID)
	if err != nil {
		fmt.Printf("on thread deleted: Error getting full message '%s': %s\n", m.ID, err)
		return
	}

	if threadTitle == "" {
		threadTitle = m.Name
	}
	if serverconfig.ActionLogChannelIgnored(m.GuildID, m.ID) || serverconfig.ActionLogChannelIgnored(m.GuildID, m.ParentID) {
		return
	}

	targetChannel := channels.Get(m.GuildID, "action-log")
	if targetChannel == "" {
		fmt.Println("No channel set up for action log")
		return
	}

	if targetChannel == m.ID || targetChannel == m.ParentID {
		return
	}

	creationTime, err := utils.CreationTime(m.ID)
	if err != nil {
		panic(err)
	}

	messageAge := time.Since(creationTime)
	if messageAge > 300*24*time.Hour {
		fmt.Printf("Skipping thread deletion log of %s since it's too old (%s)\n", m.ID, messageAge)
		return
	}

	var member *discordgo.Member
	if authorID != "unknown" {
		member, err = s.GuildMember(m.GuildID, authorID)
		if err != nil {
			fmt.Printf("Error getting guild member (%s): %s\n", authorID, err)
		}
	}

	embed := &discordgo.MessageEmbed{
		Title: "Thread deleted",
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
		Name:   "Forum/Thread",
		Value:  threadTitle,
		Inline: true,
	})
	if m.ParentID != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Parent channel",
			Value:  "<#" + m.ParentID + ">",
			Inline: true,
		})
	}
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

			targetChannel := channels.Get(m.GuildID, "moderation-action")
			if targetChannel == "" {
				fmt.Println("No channel set up for moderation actions")
				return
			}

			hidden := strings.Contains(entry.Reason, "!hide") || strings.Contains(entry.Reason, "!hidden") || strings.Contains(entry.Reason, "$hide") || strings.Contains(entry.Reason, "$hidden")

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

func postUserInfo(s *discordgo.Session, member *discordgo.Member, title string, extraFields []discordgo.MessageEmbedField) {
	embed := &discordgo.MessageEmbed{
		Title: title,
	}
	targetChannel := channels.Get(member.GuildID, "system-messages")
	if targetChannel == "" {
		fmt.Println(member.GuildID, "No channel set up for system messages")
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

	for _, extraField := range extraFields {
		embed.Fields = append(embed.Fields, &extraField)
	}

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

func onUserLeft(s *discordgo.Session, m *discordgo.GuildMemberRemove) {
	postUserInfo(s, m.Member, "User Left", nil)
}

// const weebMessageID = `552788256333234176`
const weebMessageID = `889819209507438603`
const reactionBye = "👋"

// forsen members role
// const membersRole = `825354461102473226`

func onMessageReactionAdded(s *discordgo.Session, m *discordgo.MessageReactionAdd) {
	targetChannel := channels.Get(m.GuildID, "weeb-channel")
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
	targetChannel := channels.Get(m.GuildID, "weeb-channel")
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

func handleMemberJoin(s *discordgo.Session, m *discordgo.GuildMemberAdd, sqlClient *sql.DB) {

	fields := []discordgo.MessageEmbedField{}

	autoGrantMemberRole := serverconfig.GetValue(m.GuildID, values.MemberRoleMode)
	if autoGrantMemberRole == "1" {
		if err := roles.Grant(s, m.GuildID, m.User.ID, "member"); err != nil {
			fmt.Println("Error adding Members role to user:", err)
		} else {
			fmt.Printf("Granted Members role to %s (%s)\n", m.User.Username, m.User.ID)
		}
	} else if autoGrantMemberRole == "2" {
		existingConnection, err := connection.Get(sqlClient, m.GuildID, m.User.ID)
		if err != nil {
			fmt.Println("Error getting current connection:", err)
			fields = append(fields, discordgo.MessageEmbedField{
				Name:   "Error",
				Value:  "Error getting current connection:" + err.Error(),
				Inline: false,
			})
		} else {
			if existingConnection != nil {
				// User had a connection already
				fields = append(fields, discordgo.MessageEmbedField{
					Name:   "Existing Connection",
					Value:  fmt.Sprintf("Twitch %s (%s)", existingConnection.TwitchUserLogin, existingConnection.TwitchUserID),
					Inline: false,
				})
				if err := roles.Grant(s, m.GuildID, m.User.ID, "member"); err != nil {
					fmt.Println("Error adding Members role to user:", err)
				} else {
					fmt.Printf("Granted Members role to %s (%s)\n", m.User.Username, m.User.ID)
				}
			} else {
				// TODO :DEDUPE THIS
				// User had no existing connection

				// Generate connection ID
				connectionID := connection.RegisterPendingConnection(m.User.ID, m.GuildID)

				// Put that connection ID in a memory map
				connection.RegisterPendingConnection(connectionID, m.User.ID)

				connectionURL := fmt.Sprintf("%s/login?code=%s", config.Domain, connectionID)
				privacyPolicy := fmt.Sprintf("%s/legal/privacy-policy", config.Domain)
				termsOfService := fmt.Sprintf("%s/legal/terms-of-service", config.Domain)

				manualVerificationChannel := channels.Get(m.GuildID, "manual-verification")
				messageContent := fmt.Sprintf("Hi! You need to authenticate to join the Discord. Please click [this link](<%s>) and sign in with your Twitch account to continue.\n\nPlease review our [privacy policy](<%s>) and [terms of service](<%s>) before proceeding.\n\nIf you run into any issues, ask for help in the <#%s> channel, or rejoin the server to try again. If the channel says **No Access** you are either already verified, or you are not in the server.", connectionURL, privacyPolicy, termsOfService, manualVerificationChannel)

				// Send the user a DM with a link to where they can authenticate
				// TODO: could this be done in a channel where we show a message only to them somehow?
				channel, err := s.UserChannelCreate(m.User.ID)
				if err != nil {
					fmt.Println("Something went wrong DMing", m.User.ID, err)
					fields = append(fields, discordgo.MessageEmbedField{
						Name:   "Error",
						Value:  "Failed to open DM",
						Inline: false,
					})
				} else {
					_, err := s.ChannelMessageSend(channel.ID, messageContent)
					if err != nil {
						fmt.Println("Something went wrong DMing", m.User.ID, err)
						fields = append(fields, discordgo.MessageEmbedField{
							Name:   "Error",
							Value:  fmt.Sprintf("Failed to send DM (Connection URL was `%s` was)", connectionURL),
							Inline: false,
						})
					} else {
						fields = append(fields, discordgo.MessageEmbedField{
							Name:   "ConnectionID",
							Value:  connectionID,
							Inline: false,
						})
					}
				}

			}
		}
	}

	postUserInfo(s, m.Member, "User Joined", fields)
}

func onMemberJoin(s *discordgo.Session, m *discordgo.GuildMemberAdd, sqlClient *sql.DB) {
	err := mute.ReapplyMute(s, sqlClient, m)
	if err != nil {
		fmt.Println("Error when seeing if we need to reapply mute:", err)
	}

	go func() {
		randomSeconds := 2 + rand.Intn(8)
		time.Sleep(time.Duration(randomSeconds) * time.Second)
		handleMemberJoin(s, m, sqlClient)
	}()

	// banIfUserIsYoungerThan(s, m, 1*time.Hour)

}
