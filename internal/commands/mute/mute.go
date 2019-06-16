package mute

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pajlada/pajbot2-discord/internal/config"
	"github.com/pajlada/pajbot2-discord/pkg"
	"github.com/pajlada/pajbot2-discord/pkg/commands"
	"github.com/pajlada/pajbot2-discord/pkg/utils"
	c2 "github.com/pajlada/pajbot2/pkg/commands"
	pb2utils "github.com/pajlada/pajbot2/pkg/utils"
)

func init() {
	commands.Register([]string{"$mute"}, New())
}

var _ pkg.Command = &Command{}

type Command struct {
	c2.Base
}

func New() *Command {
	return &Command{
		Base: c2.NewBase(),
	}
}

func (c *Command) Run(s *discordgo.Session, m *discordgo.MessageCreate, parts []string) (res pkg.CommandResult) {
	res = pkg.CommandResultNoCooldown
	const usage = `$mute @user duration <reason> (i.e. $mute @user 1h5m shitposting in serious channel)`

	var err error
	var targetID string
	var duration time.Duration
	var reason string

	// FIXME
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

	targetID = utils.CleanUserID(parts[0])

	if targetID == "" {
		s.ChannelMessageSend(m.ChannelID, m.Author.Mention()+" $mute invalid user. usage: "+usage)
		return
	}

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
		UserID:  targetID,
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
	err = s.GuildMemberRoleAdd(m.GuildID, targetID, config.MutedRole)
	if err != nil {
		fmt.Println("Error assigning role:", err)
	}

	// Announce mute in action channel
	s.ChannelMessageSend(config.ModerationActionChannelID, fmt.Sprintf("%s muted %s for %s. reason: %s", m.Author.Mention(), targetID, duration, reason))
	fmt.Println(config.ModerationActionChannelID, fmt.Sprintf("%s muted %s for %s. reason: %s", m.Author.Mention(), targetID, duration, reason))

	// Announce mute success
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s mute %s for %s. reason: %s", m.Author.Mention(), targetID, duration, reason))
	fmt.Println(m.ChannelID, fmt.Sprintf("%s mute %s for %s. reason: %s", m.Author.Mention(), targetID, duration, reason))

	return
}

func (c *Command) Description() string {
	return c.Base.Description
}
