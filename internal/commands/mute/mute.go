package mute

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/basecommand"
	"github.com/pajbot/pajbot2-discord/internal/config"
	"github.com/pajbot/pajbot2-discord/internal/serverconfig"
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
	res = pkg.CommandResultNoCooldown
	const usage = `$mute @user duration <reason> (i.e. $mute @user 1h5m shitposting in serious channel)`

	var err error
	var duration time.Duration
	var reason string

	hasAccess, err := utils.MemberInRoles(s, m.GuildID, m.Author.ID, config.MiniModeratorRoles)
	if err != nil {
		fmt.Println("Error:", err)
		return pkg.CommandResultUserCooldown
	}
	if !hasAccess {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s, you don't have permission to use $mute", m.Author.Mention()))
		return pkg.CommandResultUserCooldown
	}

	parts = parts[1:]

	if len(parts) < 2 {
		s.ChannelMessageSend(m.ChannelID, m.Author.Mention()+" usage: "+usage)
		return
	}

	if len(m.Mentions) == 0 {
		s.ChannelMessageSend(m.ChannelID, "missing user arg. usage: $mute <user> <time> <reason>")
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
	timepoint := time.Now().Add(duration)

	action := pkg.Action{
		Type:    "unmute",
		GuildID: m.GuildID,
		UserID:  target.ID,
		RoleID:  config.MutedRole,
	}
	bytes, err := json.Marshal(&action)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, m.Author.Mention()+" $mute unable to marshal action: "+err.Error())
		return
	}

	query := "INSERT INTO discord_queue (action, timepoint) VALUES ($1, $2)"
	_, err = commands.SQLClient.Exec(query, string(bytes), timepoint)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, m.Author.Mention()+" $mute sql error: "+err.Error())
		return
	}

	// Assign muted role
	err = s.GuildMemberRoleAdd(m.GuildID, target.ID, config.MutedRole)
	if err != nil {
		fmt.Println("Error assigning role:", err)
	}

	const resultFormat = "%s muted %s (%s - %s) for %s. reason: %s"
	resultMessage := fmt.Sprintf(resultFormat, m.Author.Mention(), target.Username, target.ID, target.Mention(), duration, reason)

	// Announce mute success in channel
	s.ChannelMessageSend(m.ChannelID, resultMessage)

	targetChannel := serverconfig.Get(m.GuildID, "channel:moderation-action")
	if targetChannel == "" {
		fmt.Println("No channel set up for moderation actions")
		return
	}

	// Announce mute in moderation-action channel
	s.ChannelMessageSend(targetChannel, resultMessage)

	return
}

func (c *Command) Description() string {
	return c.Command.Description
}
