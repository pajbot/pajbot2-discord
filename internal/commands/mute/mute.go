package mute

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/basecommand"
	"github.com/pajbot/pajbot2-discord/internal/channels"
	"github.com/pajbot/pajbot2-discord/internal/roles"
	"github.com/pajbot/pajbot2-discord/pkg"
	"github.com/pajbot/pajbot2-discord/pkg/commands"
	"github.com/pajbot/pajbot2-discord/pkg/utils"
	pb2utils "github.com/pajbot/utils"
)

func init() {
	commands.Register([]string{"$mute"}, New())
}

var _ pkg.Command = &Command{}

type Command struct {
	basecommand.Command
}

func New() *Command {
	return &Command{
		Command: basecommand.New(),
	}
}

func (c *Command) Run(s *discordgo.Session, m *discordgo.MessageCreate, parts []string) (res pkg.CommandResult) {
	// TODO: Deprecate
	res = pkg.CommandResultNoCooldown
	const usage = `$mute @user duration <reason> (i.e. $mute @user 1h5m shitposting in serious channel)`

	var err error
	var duration time.Duration
	var reason string

	hasAccess, err := utils.MemberHasPermission(s, m.GuildID, m.Author.ID, "minimod")
	if err != nil {
		fmt.Println("Error:", err)
		return pkg.CommandResultUserCooldown
	}
	if !hasAccess {
		utils.Reply(s, m, "you don't have permission to use this command")
		return pkg.CommandResultUserCooldown
	}

	mutedRole := roles.GetSingle(m.GuildID, "muted")
	if mutedRole == "" {
		utils.Reply(s, m, "no muted role has been assigned")
		return
	}

	parts = parts[1:]

	if len(parts) < 2 {
		s.ChannelMessageSend(m.ChannelID, m.Author.Mention()+" usage: "+usage)
		return
	}

	if len(m.Mentions) == 0 {
		s.ChannelMessageSend(m.ChannelID, "Missing user argument. Usage: $mute <user> <time> <reason>")
		return
	}

	target := m.Mentions[0]

	duration, err = pb2utils.ParseDuration(parts[1])
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, m.Author.Mention()+" $mute invalid duration: "+err.Error())
		return
	}

	if duration < 1*time.Minute {
		duration = 1 * time.Minute
	} else if duration > 14*24*time.Hour {
		duration = 14 * 24 * time.Hour
	}

	reason = strings.Join(parts[2:], " ")

	// Create queued up unmute action in database
	muteEnd := time.Now().Add(duration)

	query := "INSERT INTO discord_mutes (guild_id, user_id, reason, mute_start, mute_end) VALUES ($1, $2, $3, NOW(), $4) ON CONFLICT (guild_id, user_id) DO UPDATE SET reason=$3, mute_end=$4"
	_, err = commands.SQLClient.Exec(query, m.GuildID, target.ID, reason, muteEnd)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, m.Author.Mention()+" $mute sql error: "+err.Error())
		return
	}

	// Assign muted role
	err = s.GuildMemberRoleAdd(m.GuildID, target.ID, mutedRole)
	if err != nil {
		fmt.Println("Error assigning role:", err)
	}

	// TODO: result message should be different if mute was updated instead of inserted
	const resultFormat = "%s muted %s for %s. reason: %s"
	resultMessage := fmt.Sprintf(resultFormat, utils.MentionUser(s, m.GuildID, m.Author), utils.MentionUser(s, m.GuildID, target), duration, reason)

	// Announce mute success in channel
	s.ChannelMessageSend(m.ChannelID, resultMessage)

	targetChannel := channels.Get(m.GuildID, "moderation-action")
	if targetChannel != "" {
		// Announce mute in moderation-action channel
		s.ChannelMessageSend(targetChannel, resultMessage)
	}

	return
}

func (c *Command) Description() string {
	return c.Command.Description
}

func Execute(s *discordgo.Session, guildID string, moderator *discordgo.User, target *discordgo.User, durationString, reason string) (string, error) {
	var duration time.Duration
	message := ""

	hasAccess, err := utils.MemberHasPermission(s, guildID, moderator.ID, "minimod")
	if err != nil {
		return message, fmt.Errorf("permission error: %w", err)
	}
	if !hasAccess {
		return message, fmt.Errorf("you don't have permission to use this command")
	}

	mutedRole := roles.GetSingle(guildID, "muted")
	if mutedRole == "" {
		return message, fmt.Errorf("no muted role has been assigned")
	}

	duration, err = pb2utils.ParseDuration(durationString)
	if err != nil {
		return message, fmt.Errorf("invalid duration: %w", err)
	}

	if duration < 1*time.Minute {
		duration = 1 * time.Minute
	} else if duration > 14*24*time.Hour {
		duration = 14 * 24 * time.Hour
	}

	// Create queued up unmute action in database
	muteEnd := time.Now().Add(duration)

	query := "INSERT INTO discord_mutes (guild_id, user_id, reason, mute_start, mute_end) VALUES ($1, $2, $3, NOW(), $4) ON CONFLICT (guild_id, user_id) DO UPDATE SET reason=$3, mute_end=$4"
	_, err = commands.SQLClient.Exec(query, guildID, target.ID, reason, muteEnd)
	if err != nil {
		return message, fmt.Errorf("sql error: %w", err)
	}

	// Assign muted role
	err = s.GuildMemberRoleAdd(guildID, target.ID, mutedRole)
	if err != nil {
		return message, fmt.Errorf("error assigning muted role %w", err)
	}

	// TODO: result message should be different if mute was updated instead of inserted
	const resultFormat = "%s muted %s for %s. reason: %s"
	message = fmt.Sprintf(resultFormat, utils.MentionUser(s, guildID, moderator), utils.MentionUser(s, guildID, target), duration, reason)

	targetChannel := channels.Get(guildID, "moderation-action")
	if targetChannel != "" {
		// Announce mute in moderation-action channel
		s.ChannelMessageSend(targetChannel, message)
	}

	return message, nil
}
