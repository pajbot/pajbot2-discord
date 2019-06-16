package pkg

import "github.com/bwmarrin/discordgo"

// CommandResult xd
type CommandResult int

const (
	// CommandResultNoCooldown xd
	CommandResultNoCooldown CommandResult = iota
	// CommandResultUserCooldown xd
	CommandResultUserCooldown
	// CommandResultGlobalCooldown xd
	CommandResultGlobalCooldown
	// CommandResultFullCooldown xd
	CommandResultFullCooldown
)

// Command xd
type Command interface {
	HasUserIDCooldown(string) bool
	AddUserIDCooldown(string)
	AddGlobalCooldown()
	Run(s *discordgo.Session, m *discordgo.MessageCreate, parts []string) CommandResult
	Description() string
}
