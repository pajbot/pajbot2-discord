package slashcommands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func init() {
	var perms int64 = discordgo.PermissionAdministrator
	cmd := &SlashCommand{
		name: "admin",
		command: &discordgo.ApplicationCommand{
			Description: "admin utility commands",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "clear-write-perms",
					Description: "b",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
			DefaultMemberPermissions: &perms,
		},
		handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			switch i.Type {
			case discordgo.InteractionApplicationCommand:
				options := i.ApplicationCommandData().Options
				if len(options) == 0 {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "No subcommand specified",
						},
					})
					return
				}

				switch options[0].Name {
				case "clear-write-perms":
					handleClearWritePerms(s, i)

				default:
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: fmt.Sprintf("Unknown subcommand: %s", options[0].Name),
						},
					})
				}
			}
		},
		registeredCommands: []*discordgo.ApplicationCommand{},
	}

	register(cmd)
}

func handleClearWritePerms(s *discordgo.Session, i *discordgo.InteractionCreate) {
	const dryRun = false

	channel, err := s.Channel(i.ChannelID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Error getting channel: %v", err),
			},
		})
		return
	}

	updated := 0

	for _, overwrite := range channel.PermissionOverwrites {
		if overwrite.Type != discordgo.PermissionOverwriteTypeMember {
			continue
		}

		if (overwrite.Allow & discordgo.PermissionSendMessages) != 0 {
			// remove send message permission
			newAllow := overwrite.Allow & ^discordgo.PermissionSendMessages

			if dryRun {
				fmt.Println("Would have removed send message perms from", overwrite.ID)
			} else {
				fmt.Println("Removing channel send message perms from", overwrite.ID)
				err := s.ChannelPermissionSet(channel.ID, overwrite.ID, overwrite.Type, newAllow, overwrite.Deny)
				if err != nil {
					fmt.Printf("Error updating permission for %s: %v\n", overwrite.ID, err)
					continue
				}
			}

			updated++
		}
	}

	message := fmt.Sprintf("Cleared write permissions for %d user(s)", updated)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
		},
	})
}
