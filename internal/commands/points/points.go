package points

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/basecommand"
	"github.com/pajbot/pajbot2-discord/pkg"
	"github.com/pajbot/pajbot2-discord/pkg/commands"
)

func init() {
	commands.Register([]string{"$points"}, New())
}

var _ pkg.Command = &Command{}

type Command struct {
	basecommand.Command
}

type User struct {
	ID                   int64  `json:"id"`
	Name                 string `json:"username"`
	DisplayName          string `json:"username_raw"`
	Points               int64  `json:"points"`
	NlRank               int    `json:"nl_rank"`
	PointsRank           int    `json:"points_rank"`
	Level                int    `json:"level"`
	LastSeen             string `json:"last_seen"`
	LastActive           string `json:"last_active"`
	Subscriber           bool   `json:"subscriber"`
	NumLines             int    `json:"num_lines"`
	MinutesInChatOffline int    `json:"minutes_in_chat_offline"`
	MinutesInChatOnline  int    `json:"minutes_in_chat_online"`
	Banned               bool   `json:"banned"`
	Ignored              bool   `json:"ignored"`
}

func New() *Command {
	return &Command{
		Command: basecommand.New(),
	}
}

func (c *Command) Run(s *discordgo.Session, m *discordgo.MessageCreate, parts []string) (res pkg.CommandResult) {
	res = pkg.CommandResultUserCooldown

	if len(parts) < 2 {
		s.ChannelMessageSend(m.ChannelID, "You need to provide a twitch username. usage: $points <username>")
		return
	}

	target := parts[1]

	resp, err := http.Get(fmt.Sprintf("https://forsen.tv/api/v1/users/%s", target))
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "The user doesn't exist or you typed the nickname incorrectly.")
		return
	}
	defer resp.Body.Close()

	var user User

	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "There was an error requesting the data, the API didn't send any information."+err.Error())
		return
	}

	const resultFormat = "%s, user %s has %d points."
	resultMessage := fmt.Sprintf(resultFormat, m.Author.Mention(), user.DisplayName, user.Points)

	s.ChannelMessageSend(m.ChannelID, resultMessage)

	return
}

func (c *Command) Description() string {
	return c.Command.Description
}
