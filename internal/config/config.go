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

	// Guild IDs in which to enable all slash commands
	SlashCommandGuildIDs []string

	// Twitter
	TwitterConsumerKey       string
	TwitterConsumerSecret    string
	TwitterAccessToken       string
	TwitterAccessTokenSecret string
)

func init() {
	DSN = stringEnv(envName("SQL_DSN"), "postgres:///pajbot2_discord?sslmode=disable")

	Token = mustStringEnv(envName("TOKEN"))

	SlashCommandGuildIDs = cleanList(stringListEnv(envName("SLASH_COMMAND_GUILD_IDS"), []string{}))

	// Twitter
	TwitterConsumerKey = stringEnv(envName("TWITTER_CONSUMER_KEY"), "")
	TwitterConsumerSecret = stringEnv(envName("TWITTER_CONSUMER_SECRET"), "")
	TwitterAccessToken = stringEnv(envName("TWITTER_ACCESS_TOKEN"), "")
	TwitterAccessTokenSecret = stringEnv(envName("TWITTER_ACCESS_TOKEN_SECRET"), "")
}
