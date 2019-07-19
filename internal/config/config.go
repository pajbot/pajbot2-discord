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
	
	ColorPickerRoles []string
	
	ColorPickerNames []string
	ColorPickerRoleMap map[string]string
)

func init() {
	DSN = stringEnv("PAJBOT2_DISCORD_BOT_SQL_DSN", "postgres:///pajbot2_discord?sslmode=disable")

	Token = mustStringEnv("PAJBOT2_DISCORD_BOT_TOKEN")

	// roles
	MiniModeratorRole = mustStringEnv("PAJBOT2_DISCORD_BOT_MINI_MODERATOR_ROLE")
	ModeratorRole = mustStringEnv("PAJBOT2_DISCORD_BOT_MODERATOR_ROLE")
	AdminRole = mustStringEnv("PAJBOT2_DISCORD_BOT_ADMIN_ROLE")
	MutedRole = mustStringEnv("PAJBOT2_DISCORD_BOT_MUTED_ROLE")
	NitroBoosterRole = mustStringEnv("PAJBOT2_DISCORD_NITRO_BOOSTER_ROLE")
	
	// colors
	ColorRedRole = mustStringEnv("PAJBOT2_DISCORD_COLOR_RED_ROLE")
	ColorGreenRole = mustStringEnv("PAJBOT2_DISCORD_COLOR_GREEN_ROLE")
	ColorBlueRole = mustStringEnv("PAJBOT2_DISCORD_COLOR_BLUE_ROLE")

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
	
	ColorPickerRoles = []string{
		NitroBoosterRole
	}
	
	ColorPickerNames = []string{
		"red"
		"green"
		"blue"
	}
	
	ColorPickerRoleMap = map[string]string{
		"red": ColorRedRole
		"green": ColorGreenRole
		"blue": ColorBlueRole
	}
}
