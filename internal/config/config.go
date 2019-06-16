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

	ModerationActionChannelID string

	ActionLogChannelID string

	WeebChannelID string
)

func init() {
	DSN = stringEnv("PAJBOT2_DISCORD_BOT_SQL_DSN", "postgres:///pajbot2_discord?sslmode=disable")

	Token = mustStringEnv("PAJBOT2_DISCORD_BOT_TOKEN")

	ModerationActionChannelID = mustStringEnv("PAJBOT2_DISCORD_BOT_MODERATION_ACTION_CHANNEL_ID")
	ActionLogChannelID = mustStringEnv("PAJBOT2_DISCORD_BOT_ACTION_LOG_CHANNEL_ID")
	WeebChannelID = mustStringEnv("PAJBOT2_DISCORD_BOT_WEEB_CHANNEL_ID")

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
