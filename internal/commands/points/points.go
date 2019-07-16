package points

import (
	"fmt"
	"net/http"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/basecommand"
	"github.com/pajbot/pajbot2-discord/internal/config"
	"github.com/pajbot/pajbot2-discord/pkg"
	"github.com/pajbot/pajbot2-discord/pkg/commands"
	"github.com/pajbot/pajbot2-discord/pkg/utils"
)

func init() {
	commands.Register([]string{"$points"}, New())
}

var _ pkg.Command = &Command{}

type Command struct {
	basecommand.Command
}

type Points struct {
	Id   int64  `json:"id"`
	Name string `json:"username"`
	NameRaw string `json:"username_raw"`
	Points string `json:"points"`
	NlRank string `json:"nl_rank"`
	PointsRank string `json:"points_rank"`
	Level string `json:"level"`
	LastSeen string `json:"last_seen"`
	LastActive string `json:"last_active"`
	Subscriber string `json:"subscriber"`
	NumLines string `json:"num_lines"`
	MinsOnline string `json:"minutes_in_chat_online"`
	MinsOffline string `json:"minutes_in_chat_offline"`
	Banned string `json:"banned"`
	Ignored string `json:"ignored"`
}

func New() *Command {
	return &Command{
		Command: basecommand.New(),
	}
}

func (c *Command) Run(s *discordgo.Session, m *discordgo.MessageCreate, parts []string) (res pkg.CommandResult) {
	res = pkg.CommandResultNoCooldown
	
	if len(parts) < 1 {
		s.ChannelMessageSend(m.ChannelID, "You need to provide a twitch username. usage: $points <username>")
		return
	}

	target := parts[1]
	
	resp, err := http.Get("https://forsen.tv/api/v1/users/%s", target)
	defer resp.Body.Close()
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "The user doesn't exist or you typed the nickname correctly. Mind uppercases.")
		return
	}
	
	var p Points
	
	err := json.NewDecoder(resp.Body).Decode(&p)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "There was an error requesting the data, the API didn't send any information.")
		return
	}

	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("User %s has %s points.", p.NameRaw, p.Points))

	return
}

func (c *Command) Description() string {
	return c.Command.Description
}
