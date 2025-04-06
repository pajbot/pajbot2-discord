package slashcommands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func init() {
	var perms int64 = discordgo.PermissionAdministrator
	cmd := &SlashCommand{
		name: "test",
		command: &discordgo.ApplicationCommand{
			Description: "Subcommands and command groups example",
			Options: []*discordgo.ApplicationCommandOption{
				// When a command has subcommands/subcommand groups
				// It must not have top-level options, they aren't accesible in the UI
				// in this case (at least not yet), so if a command has
				// subcommands/subcommand any groups registering top-level options
				// will cause the registration of the command to fail

				{
					Name:        "subcommand-group",
					Description: "Subcommands group",
					Options: []*discordgo.ApplicationCommandOption{
						// Also, subcommand groups aren't capable of
						// containing options, by the name of them, you can see
						// they can only contain subcommands
						{
							Name:        "nested-subcommand",
							Description: "Nested subcommand",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
						},
					},
					Type: discordgo.ApplicationCommandOptionSubCommandGroup,
				},
				// Also, you can create both subcommand groups and subcommands
				// in the command at the same time. But, there's some limits to
				// nesting, count of subcommands (top level and nested) and options.
				// Read the intro of slash-commands docs on Discord dev portal
				// to get more information
				{
					Name:        "subcommand",
					Description: "Top-level subcommand",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
			DefaultMemberPermissions: &perms,
		},
		handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			switch i.Type {
			case discordgo.InteractionApplicationCommand:
				options := i.ApplicationCommandData().Options
				fmt.Println(options[0].Name)
			}
		},
		registeredCommands: []*discordgo.ApplicationCommand{},
	}

	register(cmd)
}
