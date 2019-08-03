package config

const (
	envPrefix = "PAJBOT2_DISCORD_BOT_"
)

func envName(v string) string {
	return envPrefix + v
}

var (
	DSN string

	Token string
	
	// Twitter
	TwitterConsumerKey 			string
	TwitterConsumerSecret 		string
	TwitterAccessToken			string
	TwitterAccessTokenSecret	string
)

func init() {
	DSN = stringEnv(envName("SQL_DSN"), "postgres:///pajbot2_discord?sslmode=disable")

	Token = mustStringEnv(envName("TOKEN"))
	
	// Twitter
	TwitterConsumerKey = stringEnv(envName("TWITTER_CONSUMER_KEY"), nil)
	TwitterConsumerSecret = stringEnv(envName("TWITTER_CONSUMER_SECRET"), nil)
	TwitterAccessToken = stringEnv(envName("TWITTER_ACCESS_TOKEN"), nil)
	TwitterAccessTokenSecret = stringEnv(envName("TWITTER_ACCESS_TOKEN_SECRET"), nil)
}
