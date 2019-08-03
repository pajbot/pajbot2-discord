package whereisstreamer

import (
	"encoding/json"
	"fmt"
	"net/http"
	
	"github.com/dghubble/go-twitter"
	"github.com/dghubble/oauth1"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/basecommand"
	"github.com/pajbot/pajbot2-discord/internal/config"
	"github.com/pajbot/pajbot2-discord/pkg"
	"github.com/pajbot/pajbot2-discord/pkg/commands"
)

func init() {
	commands.Register([]string{"$whereisclown"}, New())
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

func getClient()(*twitter.Client, error) {

	if config.TwitterUserName == nil || config.TwitterConsumerKey == nil || config.TwitterConsumerSecret == nil || 
		config.TwitterAccessToken == nil || config.TwitterAccessTokenSecret == nil {
		return nil, "Twitter credentials are not correctly set in the configuration."
	}

    config := oauth1.NewConfig(config.TwitterConsumerKey, config.TwitterConsumerSecret)
    token := oauth1.NewToken(config.TwitterAccessToken, config.TwitterAccessTokenSecret)

    httpClient := config.Client(oauth1.NoContext, token)
    client := twitter.NewClient(httpClient)

    // Verify Credentials
    verifyParams := &twitter.AccountVerifyParams{
        SkipStatus:   twitter.Bool(true),
        IncludeEmail: twitter.Bool(true),
    }

    // verify
    user, _, err := client.Accounts.VerifyCredentials(verifyParams)
    if err != nil {
        return nil, err
    }

    return client, nil
}

func (c *Command) Run(s *discordgo.Session, m *discordgo.MessageCreate, parts []string) (res pkg.CommandResult) {
	res = pkg.CommandResultUserCooldown
	
	client, err := getClient()
	
	if err != nil {
		log.Println("Error getting Twitter Client. Verify credentials configuration.")
		return
	}
	
	tweets, resp, err := client.Timelines.UserTimeline(&twitter.UserTimelineParams{
		ScreenName: config.TwitterUserName,
		ExcludeReplies: true,
		IncludeRetweets: false,
		Count: 1
	})

	if err != nil {
		log.Println("Error getting Tweets from specified user. Verify credentials configuration.")
		log.Println(err)
		return
	}
	
	if len(tweets) < 1 {
		s.ChannelMessageSend(m.ChannelID, "There was an error requesting the tweet information.")
		return
	}

	const resultFormat = "Last tweet from clown: https://twitter.com/%s/status/%s"
	resultMessage := fmt.Sprintf(resultFormat, config.TwitterUserName, tweets[0].id)

	s.ChannelMessageSend(m.ChannelID, resultMessage)

	return
}

func (c *Command) Description() string {
	return c.Command.Description
}
