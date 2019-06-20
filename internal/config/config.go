package config

var (
	MiniModeratorRole string
	ModeratorRole     string
	AdminRole         string
	MutedRole         string

	DSN string

	Token string

	AdminRoles []string

	ModeratorRoles []string

	MiniModeratorRoles []string
)

func init() {
	DSN = stringEnv("PAJBOT2_DISCORD_BOT_SQL_DSN", "postgres:///pajbot2_discord?sslmode=disable")

	Token = mustStringEnv("PAJBOT2_DISCORD_BOT_TOKEN")

	// roles
	MiniModeratorRole = mustStringEnv("PAJBOT2_DISCORD_BOT_MINI_MODERATOR_ROLE")
	ModeratorRole = mustStringEnv("PAJBOT2_DISCORD_BOT_MODERATOR_ROLE")
	AdminRole = mustStringEnv("PAJBOT2_DISCORD_BOT_ADMIN_ROLE")
	MutedRole = mustStringEnv("PAJBOT2_DISCORD_BOT_MUTED_ROLE")

	AdminRoles = []string{
		AdminRole,
	}

	ModeratorRoles = []string{
		AdminRole,
		ModeratorRole,
	}

	MiniModeratorRoles = []string{
		AdminRole,
		ModeratorRole,
		MiniModeratorRole,
	}
}
