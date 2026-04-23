package slashcommands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/pajbot2-discord/internal/channelrole"
	"github.com/pajbot/pajbot2-discord/pkg/utils"
)

func init() {
	cmd := &SlashCommand{
		name: "unsubscribe",

		command: &discordgo.ApplicationCommand{
			Description: "Unsubscribe from a channel topic role",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionRole,
					Name:        "role",
					Description: "Role to unsubscribe from",
					Required:    true,
				},
			},
		},

		handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			var executingUser *discordgo.User

			if i.Member != nil {
				executingUser = i.Member.User
			} else if i.User != nil {
				executingUser = i.User
			} else {
				fmt.Println("No user found for unsubscribe?")
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "No user found for unsubscribe?",
					},
				})
				return
			}

			options := i.ApplicationCommandData().Options
			roleToUnsubscribe := options[0].RoleValue(s, i.GuildID)
			if roleToUnsubscribe == nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:   discordgo.MessageFlagsEphemeral,
						Content: "An invalid role was passed to the subscribe command",
					},
				})
				return
			}

			userInRole, err := utils.MemberInRoles(s, i.GuildID, executingUser.ID, roleToUnsubscribe.ID)
			if err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:   discordgo.MessageFlagsEphemeral,
						Content: "Error checking role status:" + err.Error(),
					},
				})
				return
			}

			if !userInRole {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:   discordgo.MessageFlagsEphemeral,
						Content: "You are not subscribed to this role",
					},
				})
				return
			}

			// Check if the role is a channel topic role, otherwise we won't allow them to unsubscribe from it.
			isChannelRole, _ := channelrole.IsChannelRole(i.GuildID, roleToUnsubscribe.ID)
			if !isChannelRole {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:   discordgo.MessageFlagsEphemeral,
						Content: "This role is not a channel topic role",
					},
				})
				return
			}

			err = s.GuildMemberRoleRemove(i.GuildID, executingUser.ID, roleToUnsubscribe.ID)
			if err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:   discordgo.MessageFlagsEphemeral,
						Content: "Error removing role:" + err.Error(),
					},
				})
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags:   discordgo.MessageFlagsEphemeral,
					Content: fmt.Sprintf("You are now unsubscribed from %s", roleToUnsubscribe.Name),
				},
			})
		},
	}

	register(cmd)
}
