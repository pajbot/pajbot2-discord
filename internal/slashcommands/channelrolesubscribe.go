package slashcommands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
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
				fmt.Println("no user found for self-mute?")
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "no user found for self-mute?",
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
						Content: "you are already subscribed to this role",
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
