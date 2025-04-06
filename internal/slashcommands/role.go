package slashcommands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/pajbot2-discord/internal/roles"
)

func init() {
	var perms int64 = discordgo.PermissionAdministrator
	cmd := &SlashCommand{
		name: "role",
		command: &discordgo.ApplicationCommand{
			Description: "Configure role",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "clear",
					Description: "Clear role",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "bot-role",
							Description: "Bot role",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
							Choices:     roles.Choices(),
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "get",
					Description: "Get information about a specific role",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "bot-role",
							Description: "Bot role",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
							Choices:     roles.Choices(),
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "set",
					Description: "Set role",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "bot-role",
							Description: "Bot role",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
							Choices:     roles.Choices(),
						},
						{
							Name:        "discord-role",
							Description: "Discord role",
							Type:        discordgo.ApplicationCommandOptionRole,
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "list",
					Description: "List roles",
				},
			},
			DefaultMemberPermissions: &perms,
		},
		handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			switch i.Type {
			case discordgo.InteractionApplicationCommand:
				options := i.ApplicationCommandData().Options
				subcommand := options[0]
				switch subcommand.Name {
				case "set":
					roleSet(s, i, subcommand.Options)
				case "get":
					roleGet(s, i, subcommand.Options)
				case "clear":
					roleClear(s, i, subcommand.Options)
				case "list":
					roleList(s, i)
				}
			}
		},
		registeredCommands: []*discordgo.ApplicationCommand{},
	}

	register(cmd)
}

func roleSet(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	botRole := options[0].StringValue()
	discordRoleID := options[1].RoleValue(s, i.GuildID).ID

	if err := roles.Set(sqlClient, i.GuildID, botRole, discordRoleID); err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Error setting %s role to <@&%s>: %s", botRole, discordRoleID, err),
			},
		})
	} else {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Set role %s to <@&%s>", botRole, discordRoleID),
			},
		})
	}
}

func formatRole(botRole, discordRoleID string) string {
	if discordRoleID == "" {
		return fmt.Sprintf("%s: No Discord role", botRole)
	}

	return fmt.Sprintf("%s: <@&%s>", botRole, discordRoleID)
}

func roleGet(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	botRole := options[0].StringValue()

	discordRoleID := roles.GetSingle(i.GuildID, botRole)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Role info: %s", formatRole(botRole, discordRoleID)),
		},
	})
}

func roleClear(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	botRole := options[0].StringValue()

	if err := roles.Clear(sqlClient, i.GuildID, botRole); err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Error clearing %s role: %s", botRole, err),
			},
		})
	} else {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Cleared role %s", botRole),
			},
		})
	}
}

func roleList(s *discordgo.Session, i *discordgo.InteractionCreate) {
	response := ""
	first := true

	for role := range roles.List() {
		if !first {
			response += "\n"
		}

		discordRoleID := roles.GetSingle(i.GuildID, role.Name)

		response += formatRole(role.Name, discordRoleID)

		first = false
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: response,
		},
	})
}
