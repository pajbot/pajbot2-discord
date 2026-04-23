package slashcommands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/pajbot2-discord/internal/channelrole"
)

func init() {
	var perms int64 = discordgo.PermissionManageRoles
	cmd := &SlashCommand{
		name: "deletechannelrole",

		command: &discordgo.ApplicationCommand{
			Description: "Deletes a channel role",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionRole,
					Name:        "role",
					Description: "Role to delete",
					Required:    true,
				},
			},
			DefaultMemberPermissions: &perms,
		},

		handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			var moderator *discordgo.User

			if i.Member != nil {
				moderator = i.Member.User
			} else if i.User != nil {
				moderator = i.User
			} else {
				fmt.Println("no moderator found?")
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "no moderator found?",
					},
				})
				return
			}

			options := i.ApplicationCommandData().Options
			roleToDelete := options[0].RoleValue(s, i.GuildID)

			if roleToDelete == nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:   discordgo.MessageFlagsEphemeral,
						Content: "an invalid role was passed to the delete channel role command",
					},
				})
				return
			}

			err := channelrole.Delete(s, moderator, i.GuildID, roleToDelete.ID, roleToDelete.Name)
			if err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:   discordgo.MessageFlagsEphemeral,
						Content: "Error deleting channel role:" + err.Error(),
					},
				})
			}

			s.GuildRoleDelete(i.GuildID, roleToDelete.ID)
			if err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:   discordgo.MessageFlagsEphemeral,
						Content: "Error deleting channel role:" + err.Error(),
					},
				})
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags:   discordgo.MessageFlagsEphemeral,
					Content: fmt.Sprintf("Channel role %s deleted", roleToDelete.Name),
				},
			})
		},
	}

	register(cmd)
}
