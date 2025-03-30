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

	WebListenAddr string

	Domain string

	// Twitter
	TwitterConsumerKey       string
	TwitterConsumerSecret    string
	TwitterAccessToken       string
	TwitterAccessTokenSecret string

	// Twitch
	TwitchClientID     string
	TwitchClientSecret string
	TwitchRedirectURI  string
)

func init() {
	DSN = stringEnv(envName("SQL_DSN"), "postgres:///pajbot2_discord?sslmode=disable")

	Token = mustStringEnv(envName("TOKEN"))

	SlashCommandGuildIDs = cleanList(stringListEnv(envName("SLASH_COMMAND_GUILD_IDS"), []string{}))

	WebListenAddr = stringEnv(envName("WEB_LISTEN_ADDR"), "127.0.0.1:7072")

	Domain = mustStringEnv(envName("DOMAIN"))

	// Twitter
	TwitterConsumerKey = stringEnv(envName("TWITTER_CONSUMER_KEY"), "")
	TwitterConsumerSecret = stringEnv(envName("TWITTER_CONSUMER_SECRET"), "")
	TwitterAccessToken = stringEnv(envName("TWITTER_ACCESS_TOKEN"), "")
	TwitterAccessTokenSecret = stringEnv(envName("TWITTER_ACCESS_TOKEN_SECRET"), "")

	// Twitch
	TwitchClientID = mustStringEnv(envName("TWITCH_CLIENT_ID"))
	TwitchClientSecret = mustStringEnv(envName("TWITCH_CLIENT_SECRET"))
	TwitchRedirectURI = mustStringEnv(envName("TWITCH_REDIRECT_URI"))
}
