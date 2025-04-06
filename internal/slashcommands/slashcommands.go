package slashcommands

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

var (
	sqlClient *sql.DB
)

func Initialize(sqlClient_ *sql.DB) {
	sqlClient = sqlClient_
}

type SlashCommand struct {
	name    string
	command *discordgo.ApplicationCommand

	handler func(*discordgo.Session, *discordgo.InteractionCreate)

	registeredCommands []*discordgo.ApplicationCommand
}

type SlashCommands struct {
	guildIDs []string
}

var commands = map[string]*SlashCommand{}

// register registers a slash command to be created on the configured guilds
// read the code or the /ping command to see how the command should be created
// we will panic if something is misconfigured
func register(cmd *SlashCommand) {
	if cmd.name == "" {
		log.Fatal("Command must have a name")
	}

	if cmd.command == nil {
		log.Fatalf("[%s] Command must have `command` set", cmd.name)
	}

	if len(cmd.registeredCommands) != 0 {
		log.Fatalf("[%s] Command must NOT have any `registeredCommands` set", cmd.name)
	}

	if cmd.handler == nil {
		log.Fatalf("[%s] Command must have `handler` set", cmd.name)
	}

	if cmd.command.Name != "" {
		log.Fatalf("[%s] `command.Name` must not be set", cmd.name)
	}
	cmd.command.Name = cmd.name

	if _, ok := commands[cmd.name]; ok {
		log.Fatalf("[%s] Command with the name '%s' has already been registered", cmd.name, cmd.name)
	}

	commands[cmd.name] = cmd
}

func onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if cmd, ok := commands[i.ApplicationCommandData().Name]; ok {
		cmd.handler(s, i)
	}
}

// Create registers all available slash commands in the guilds listed in the config
func (s *SlashCommands) Create(session *discordgo.Session) error {
	session.AddHandler(onInteractionCreate)

	for _, guildID := range s.guildIDs {
		log.Println("Creating slash commands for guild", guildID)

		for _, cmd := range commands {
			registeredCommand, err := session.ApplicationCommandCreate(session.State.User.ID, guildID, cmd.command)
			if err != nil {
				return fmt.Errorf("creating command '%s' failed: %w", cmd.name, err)
			}
			cmd.registeredCommands = append(cmd.registeredCommands, registeredCommand)
		}
	}

	return nil
}

// Delete deletes all slash commands that were registered in all guilds
func (s *SlashCommands) Delete(session *discordgo.Session) error {
	for _, cmd := range commands {
		for _, registeredCommand := range cmd.registeredCommands {
			err := session.ApplicationCommandDelete(session.State.User.ID, registeredCommand.GuildID, registeredCommand.ID)
			if err != nil {
				log.Printf("Error deleting command %v: %s", cmd, err)
			}
		}
	}

	return nil
}

// New creates a SlashCommands struct with the given options
func New(guildIDs []string) *SlashCommands {
	return &SlashCommands{
		guildIDs: guildIDs,
	}
}
