package slashcommands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/pajbot2-discord/internal/channelrole"
	"github.com/pajbot/pajbot2-discord/pkg/utils"
)

func init() {
	cmd := &SlashCommand{
		name: "at",

		command: &discordgo.ApplicationCommand{
			Description: "Mention a topic role inside a channel",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionRole,
					Name:        "role",
					Description: "Role to mention",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "message",
					Description: "Your beautiful message",
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
				fmt.Println("no user found for at comnmand")
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "no user found for at command",
					},
				})
				return
			}

			options := i.ApplicationCommandData().Options
			roleToMention := options[0].RoleValue(s, i.GuildID)
			message := options[1].StringValue()

			if roleToMention == nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "an invalid role was passed to the at command",
					},
				})
				return
			}

			if message == "" {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "an invalid message was passed to the at command",
					},
				})
				return
			}

			canPing, errorMessage := channelrole.CanChannelRolePing(s, i.GuildID, executingUser.ID, i.ChannelID, roleToMention.ID)

			if !canPing && errorMessage != "" {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:   discordgo.MessageFlagsEphemeral,
						Content: errorMessage,
					},
				})
				return
			}

			err := channelrole.UpdateLastInvoked(s, executingUser, i.GuildID, roleToMention.ID, roleToMention.Name)
			if err != nil {
				fmt.Println("Error updating last invoked channel role ping:", err)
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("*from %s to %s*\n %s", utils.MentionUser(s, i.GuildID, executingUser), roleToMention.Mention(), message),
				},
			})
		},
	}

	register(cmd)
}
