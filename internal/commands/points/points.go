package points

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/basecommand"
	"github.com/pajbot/pajbot2-discord/internal/serverconfig"
	"github.com/pajbot/pajbot2-discord/internal/values"
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
	DisplayName string `json:"name"`
	Points      int64  `json:"points"`
	PointsRank  int64  `json:"points_rank"`
}

func New() *Command {
	return &Command{
		Command: basecommand.New(),
	}
}

func (c *Command) Run(s *discordgo.Session, m *discordgo.MessageCreate, parts []string) (res pkg.CommandResult) {
	res = pkg.CommandResultUserCooldown

	pajbotHost := serverconfig.GetValue(m.GuildID, values.PajbotHost)

	if pajbotHost == "" {
		s.ChannelMessageSend(m.ChannelID, "This server does not have a pajbot host configured. Use the `$configure value` command to configure the `pajbot_host` key")
		return pkg.CommandResultFullCooldown
	}

	if len(parts) < 2 {
		s.ChannelMessageSend(m.ChannelID, "You need to provide a twitch username. usage: `$points <username>`")
		return
	}

	target := parts[1]

	apiURL := url.URL{
		Scheme:   "https",
		Host:     pajbotHost,
		Path:     fmt.Sprintf("api/v1/users/%s", target),
		RawQuery: "user_input=true",
	}

	resp, err := http.Get(apiURL.String())
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "There was an error getting that user's data: "+err.Error())
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		s.ChannelMessageSend(m.ChannelID, "The user doesn't exist or you typed the nickname incorrectly.")
		return
	}

	if resp.StatusCode >= 400 {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("There was an error getting that user's data: The API returned status code %d", resp.StatusCode))
		return
	}

	var user User

	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "There was an error parsing the response from the API: "+err.Error())
		return
	}

	const resultFormat = "%s, user %s has %d points and is ranked %d."
	resultMessage := fmt.Sprintf(resultFormat, m.Author.Mention(), user.DisplayName, user.Points, user.PointsRank)

	s.ChannelMessageSend(m.ChannelID, resultMessage)

	return
}

func (c *Command) Description() string {
	return c.Command.Description
}
