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
)

func init() {
	DSN = stringEnv(envName("SQL_DSN"), "postgres:///pajbot2_discord?sslmode=disable")

	Token = mustStringEnv(envName("TOKEN"))
}
