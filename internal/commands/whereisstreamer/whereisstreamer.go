package whereisstreamer

import (
	"errors"
	"fmt"
	"log"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/basecommand"
	"github.com/pajbot/pajbot2-discord/internal/config"
	"github.com/pajbot/pajbot2-discord/internal/serverconfig"
	"github.com/pajbot/pajbot2-discord/pkg"
	"github.com/pajbot/pajbot2-discord/pkg/commands"
)

func init() {
	commands.Register([]string{"$whereisclown", "$whereisstreamer"}, New())
}

var _ pkg.Command = &Command{}

type Command struct {
	basecommand.Command

	client *twitter.Client
}

func New() *Command {
	c := &Command{
		Command: basecommand.New(),
	}

	client, err := getClient()
	if err != nil {
		return nil
	}

	c.client = client

	return c
}

func getClient() (*twitter.Client, error) {
	if config.TwitterConsumerKey == "" || config.TwitterConsumerSecret == "" ||
		config.TwitterAccessToken == "" || config.TwitterAccessTokenSecret == "" {
		return nil, errors.New("twitter credentials are not correctly set in the configuration")
	}

	oauthConfig := oauth1.NewConfig(config.TwitterConsumerKey, config.TwitterConsumerSecret)
	token := oauth1.NewToken(config.TwitterAccessToken, config.TwitterAccessTokenSecret)

	httpClient := oauthConfig.Client(oauth1.NoContext, token)
	client := twitter.NewClient(httpClient)

	// Verify Credentials
	verifyParams := &twitter.AccountVerifyParams{
		SkipStatus:   twitter.Bool(true),
		IncludeEmail: twitter.Bool(true),
	}

	// verify
	_, _, err := client.Accounts.VerifyCredentials(verifyParams)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (c *Command) Run(s *discordgo.Session, m *discordgo.MessageCreate, parts []string) (res pkg.CommandResult) {
	res = pkg.CommandResultUserCooldown

	twitterUsername := serverconfig.Get(m.GuildID, "twitter:username")
	if twitterUsername == "" {
		return
	}

	tweets, _, err := c.client.Timelines.UserTimeline(&twitter.UserTimelineParams{
		ScreenName:      twitterUsername,
		ExcludeReplies:  twitter.Bool(true),
		IncludeRetweets: twitter.Bool(false),
		Count:           1,
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

	const resultFormat = "Last tweet from clown: https://twitter.com/%s/status/%d"
	resultMessage := fmt.Sprintf(resultFormat, twitterUsername, tweets[0].ID)

	s.ChannelMessageSend(m.ChannelID, resultMessage)

	return
}

func (c *Command) Description() string {
	return c.Command.Description
}
