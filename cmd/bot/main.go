package main

import (
	"context"
	cryptorand "crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/pajbot2-discord/internal/channels"
	"github.com/pajbot/pajbot2-discord/internal/config"
	"github.com/pajbot/pajbot2-discord/internal/connection"
	"github.com/pajbot/pajbot2-discord/internal/filter"
	"github.com/pajbot/pajbot2-discord/internal/mute"
	"github.com/pajbot/pajbot2-discord/internal/roles"
	"github.com/pajbot/pajbot2-discord/internal/serverconfig"
	"github.com/pajbot/pajbot2-discord/internal/slashcommands"
	"github.com/pajbot/pajbot2-discord/internal/twitchstreamannouncer"
	"github.com/pajbot/pajbot2-discord/internal/values"
	"github.com/pajbot/pajbot2-discord/pkg"
	"github.com/pajbot/pajbot2-discord/pkg/commands"
	"github.com/pajbot/pajbot2-discord/pkg/utils"
	sharedutils "github.com/pajbot/utils"
	normalize "github.com/pajlada/lidl-normalize"
	"github.com/pajlada/stupidmigration"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/twitch"

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

type pendingConnection struct {
	connectionID   string
	discordUserID  string
	discordGuildID string
	state          string
}

var approvalForceTwitch = oauth2.SetAuthURLParam("force_verify", "true")

var connectionIDMutex = sync.Mutex{}
var connectionIDs = map[string]*pendingConnection{}

// registerPendingConnection returns the connection ID for this user
func registerPendingConnection(discordUserID, discordGuildID string) string {
	connectionIDMutex.Lock()
	defer connectionIDMutex.Unlock()
	connectionID, err := utils.GenerateRandomStringURLSafe(12)
	if err != nil {
		panic(err)
	}

	var tokenBytes [255]byte
	if _, err := cryptorand.Read(tokenBytes[:]); err != nil {
		panic(err)
	}

	connectionIDs[connectionID] = &pendingConnection{
		connectionID:   connectionID,
		discordUserID:  discordUserID,
		discordGuildID: discordGuildID,
		state:          hex.EncodeToString(tokenBytes[:]),
	}

	return connectionID
}

func getPendingConnection(connectionID string) *pendingConnection {
	connectionIDMutex.Lock()
	defer connectionIDMutex.Unlock()
	return connectionIDs[connectionID]
}
func getPendingConnectionByState(state string) (string, *pendingConnection) {
	connectionIDMutex.Lock()
	defer connectionIDMutex.Unlock()
	for connectionID, pendingConnection := range connectionIDs {
		if pendingConnection.state == state {
			return connectionID, pendingConnection
		}
	}
	return "", nil
}

func unregisterPendingConnection(connectionID string) {
	connectionIDMutex.Lock()
	defer connectionIDMutex.Unlock()
	delete(connectionIDs, connectionID)
}

type twitchValidateResponse struct {
	Login  string `json:"login"`
	UserID string `json:"user_id"`
}

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

	slashcommands.Initialize(sqlClient)
}

type App struct {
	bot *discordgo.Session

	filterRunner *filter.Runner
}

func (a *App) registerUser(pendingConnection *pendingConnection, twitchUser twitchValidateResponse) {
	defer unregisterPendingConnection(pendingConnection.connectionID)
	membersRole := roles.GetSingle(pendingConnection.discordGuildID, "member")
	if membersRole == "" {
		fmt.Println("Error applying members role: There's no members role in the server ??")
		return
	}

	if err := a.bot.GuildMemberRoleAdd(pendingConnection.discordGuildID, pendingConnection.discordUserID, membersRole); err != nil {
		fmt.Println("Error adding members role to user:", pendingConnection, twitchUser, err)
	}

	existingConnection, err := connection.Get(sqlClient, pendingConnection.discordGuildID, pendingConnection.discordUserID)
	if err != nil {
		fmt.Println("error getting existing connection:", pendingConnection, twitchUser, err)
		return
	}

	member, err := a.bot.GuildMember(pendingConnection.discordGuildID, pendingConnection.discordUserID)
	if err != nil {
		fmt.Println("error getting member:", pendingConnection, twitchUser, err)
		return
	}

	if err := connection.Upsert(sqlClient, pendingConnection.discordGuildID, pendingConnection.discordUserID, member.User.Username, twitchUser.UserID, twitchUser.Login); err != nil {
		fmt.Println("Error upserting into db:", pendingConnection, twitchUser, err)
		return
	}

	a.postConnectionUpdate(member, existingConnection, twitchUser)

	if err := roles.Grant(a.bot, pendingConnection.discordGuildID, pendingConnection.discordUserID, "member"); err != nil {
		fmt.Println("Error adding Members role to user:", err)
	} else {
		fmt.Printf("Granted Members role to %s (%s)\n", member.User.Username, member.User.ID)
	}
}

func formatConnection(connection *connection.DiscordConnection) string {
	if connection == nil {
		panic("connection may not be nil in formatConnection")
	}

	return fmt.Sprintf("`%s` (TUID `%s`)", utils.EscapeCodeBlock(connection.TwitchUserLogin), connection.TwitchUserID)
}

func (a *App) postConnectionUpdate(member *discordgo.Member, existingConnection *connection.DiscordConnection, twitchUser twitchValidateResponse) {
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
			Value:  formatConnection(existingConnection),
			Inline: false,
		})
	}

	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "New connection",
		Value:  fmt.Sprintf("`%s` (TUID `%s`)", utils.EscapeCodeBlock(twitchUser.Login), twitchUser.UserID),
		Inline: false,
	})

	altConnections, err := connection.GetByTwitchID(sqlClient, twitchUser.UserID, member.GuildID, member.User.ID)
	if err != nil {
		fmt.Println("Error getting alt connections:", err)
	} else {
		if len(altConnections) > 0 {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "Alt Detection",
				Value:  formatAltConnections(altConnections),
				Inline: false,
			})
		}
	}

	if _, err := a.bot.ChannelMessageSendEmbed(targetChannel, embed); err != nil {
		fmt.Println("Error sending message embed:", err)
	}
}

func formatAltConnections(altConnections []*connection.DiscordConnection) string {
	var b strings.Builder

	first := true

	for _, c := range altConnections {
		if !first {
			b.WriteString(", ")
		}

		b.WriteString(fmt.Sprintf("did: `%s`, dname: `%s`", c.DiscordUserID, utils.EscapeCodeBlock(c.DiscordUserName)))

		first = false
	}

	return b.String()
}

func (a *App) startWebServer(ctx context.Context) error {
	scopes := []string{}

	oauth2Config := &oauth2.Config{
		ClientID:     config.TwitchClientID,
		ClientSecret: config.TwitchClientSecret,
		Scopes:       scopes,
		Endpoint:     twitch.Endpoint,
		RedirectURL:  config.TwitchRedirectURI,
	}

	// Redirect user to twitch page
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		connectionID := r.FormValue("code")
		if connectionID == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "missing connection ID")
			return
		}

		// Validate code
		pendingConnection := getPendingConnection(connectionID)
		if pendingConnection == nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "bad connection ID")
			return
		}

		fmt.Println("Redirecting login for", pendingConnection)

		http.Redirect(w, r, oauth2Config.AuthCodeURL(pendingConnection.state, approvalForceTwitch), http.StatusTemporaryRedirect)
	})

	http.HandleFunc("/login/authorized", func(w http.ResponseWriter, r *http.Request) {
		state := r.FormValue("state")
		if state == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "missing state")
			return
		}

		// Validate state
		connectionID, pendingConnection := getPendingConnectionByState(state)
		if pendingConnection == nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "bad state")
			return
		}

		token, err := oauth2Config.Exchange(ctx, r.FormValue("code"))
		if err != nil {
			fmt.Println("Error exchanging code", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "internal server error")
			unregisterPendingConnection(connectionID)
			return
		}

		go func() {
			// Fetch user ID
			req, err := http.NewRequest("GET", "https://id.twitch.tv/oauth2/validate", nil)
			if err != nil {
				fmt.Println("error making request to validate twitch token", err)
				unregisterPendingConnection(connectionID)
				return
			}

			req.Header.Add("Authorization", fmt.Sprintf("OAuth %s", token.AccessToken))

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Println("error in request to validate twitch token", err)
				unregisterPendingConnection(connectionID)
				return
			}

			structuredResponse := twitchValidateResponse{}

			if err := json.NewDecoder(res.Body).Decode(&structuredResponse); err != nil {
				fmt.Println("error decoding validate twitch token response:", err)
				unregisterPendingConnection(connectionID)
				return
			}

			go a.registerUser(pendingConnection, structuredResponse)
		}()
	})

	go func() {
		if err := http.ListenAndServe(config.WebListenAddr, nil); err != nil {
			fmt.Println("Error in http listen and serve:", err)
		}
	}()

	return nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	<-rdyHelixClient
	fmt.Println("Twitch API initialized")

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

	app.filterRunner.AddFilter(&filter.Delete{
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

	app.filterRunner.AddFilter(&filter.Delete{
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

	app.filterRunner.AddFilter(&filter.Delete{
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

	app.filterRunner.AddFilter(&filter.DeleteAdvanced{
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
	bot.AddHandler(onUserLeft)
	bot.AddHandler(onMessageReactionAdded)
	bot.AddHandler(onMessageReactionRemoved)
	bot.AddHandler(func(s *discordgo.Session, m *discordgo.GuildCreate) {
		// Check streamer's live status occasionally and post it in the stream-status channel on updates
		go twitchstreamannouncer.Start(ctx, m.Guild.ID, helixClient, bot, sqlClient)
	})
	bot.AddHandler(func(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
		onMemberJoin(s, m, sqlClient)
	})

	if err := app.startWebServer(ctx); err != nil {
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

	go app.filterRunner.Run(ctx)

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

			targetChannel := channels.Get(m.GuildID, "action-log")
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
				Value:  strings.Replace(m.Content, "`", "", -1),
				Inline: true,
			})
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "Channel",
				Value:  "<#" + m.ChannelID + ">",
				Inline: true,
			})

			targetChannel := channels.Get(m.GuildID, "action-log")
			if targetChannel == "" {
				fmt.Println("No channel set up for moderation actions")
				return
			}

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

	autoReactIDs := serverconfig.GetAutoReact(m.GuildID, m.ChannelID)
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
		fmt.Printf("on message edit: Error getting full message '%s': %s\n", m.ID, err)
		return
	}
	targetChannel := channels.Get(m.GuildID, "action-log")
	if targetChannel == "" {
		fmt.Println("No channel set up for action log")
		return
	}

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

			targetChannel := channels.Get(m.GuildID, "action-log")
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

	targetChannel := channels.Get(m.GuildID, "action-log")
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
const forsenServerID = `97034666673975296`

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

func onMemberJoin(s *discordgo.Session, m *discordgo.GuildMemberAdd, sqlClient *sql.DB) {
	err := mute.ReapplyMute(s, sqlClient, m)
	if err != nil {
		fmt.Println("Error when seeing if we need to reapply mute:", err)
	}

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
				// User had no existing connection

				// Generate connection ID
				connectionID := registerPendingConnection(m.User.ID, m.GuildID)

				// Put that connection ID in a memory map
				registerPendingConnection(connectionID, m.User.ID)

				// TODO: Generate URL based on config stuff
				connectionURL := fmt.Sprintf("%s/login?code=%s", config.Domain, connectionID)

				manualVerificationChannel := channels.Get(m.GuildID, "manual-verification")
				messageContent := fmt.Sprintf("Hi! You need to authenticate to join the Discord. Please click [this link](<%s>) and sign in with your Twitch account to continue.\n\nIf you run into any issues, ask for help in the <#%s> channel, or rejoin the server to try again.", connectionURL, manualVerificationChannel)

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
							Value:  "Failed to send DM",
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

	// banIfUserIsYoungerThan(s, m, 1*time.Hour)

}
