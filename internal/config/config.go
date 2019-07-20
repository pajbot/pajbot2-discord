package config

const (
	envPrefix = "PAJBOT2_DISCORD_BOT_"
)

func envName(v string) string {
	return envPrefix + v
}

var (
	MiniModeratorRole string
	ModeratorRole     string
	AdminRole         string
	MutedRole         string
	NitroBoosterRole  string

	DSN string

	Token string

	AdminRoles []string

	ModeratorRoles []string

	MiniModeratorRoles []string

	ColorPickerRoles []string
)

func init() {
	DSN = stringEnv(envName("SQL_DSN"), "postgres:///pajbot2_discord?sslmode=disable")

	Token = mustStringEnv(envName("TOKEN"))

	// roles
	MiniModeratorRole = mustStringEnv(envName("MINI_MODERATOR_ROLE"))
	ModeratorRole = mustStringEnv(envName("MODERATOR_ROLE"))
	AdminRole = mustStringEnv(envName("ADMIN_ROLE"))
	MutedRole = mustStringEnv(envName("MUTED_ROLE"))
	NitroBoosterRole = mustStringEnv(envName("NITRO_BOOSTER_ROLE"))

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
		NitroBoosterRole,
	}
}
