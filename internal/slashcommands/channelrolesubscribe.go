package slashcommands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/pajbot2-discord/internal/channelrole"
	"github.com/pajbot/pajbot2-discord/pkg/utils"
)

func init() {
	cmd := &SlashCommand{
		name: "subscribe",

		command: &discordgo.ApplicationCommand{
			Description: "Subscribe to a channel topic role",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionRole,
					Name:        "role",
					Description: "Role to subscribe to",
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
				fmt.Println("No user found for subscribe?")
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "No user found for subscribe?",
					},
				})
				return
			}

			options := i.ApplicationCommandData().Options
			roleToSubscribe := options[0].RoleValue(s, i.GuildID)
			if roleToSubscribe == nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:   discordgo.MessageFlagsEphemeral,
						Content: "an invalid role was passed to the subscribe command",
					},
				})
				return
			}

			userInRole, err := utils.MemberInRoles(s, i.GuildID, executingUser.ID, roleToSubscribe.ID)
			if err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:   discordgo.MessageFlagsEphemeral,
						Content: "error checking role status:" + err.Error(),
					},
				})
				return
			}

			if userInRole {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:   discordgo.MessageFlagsEphemeral,
						Content: "You are already subscribed to this role",
					},
				})
				return
			}

			// Check if the role is a channel topic role, otherwise we won't allow them to subscribe to it.
			isChannelRole, _ := channelrole.IsChannelRole(i.GuildID, roleToSubscribe.ID)
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

			err = s.GuildMemberRoleAdd(i.GuildID, executingUser.ID, roleToSubscribe.ID)
			if err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:   discordgo.MessageFlagsEphemeral,
						Content: "error adding role:" + err.Error(),
					},
				})
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags:   discordgo.MessageFlagsEphemeral,
					Content: fmt.Sprintf("You are now subscribed to %s", roleToSubscribe.Name),
				},
			})
		},
	}

	register(cmd)
}
