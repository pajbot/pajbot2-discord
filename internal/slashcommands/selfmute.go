package slashcommands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/pajbot2-discord/internal/channels"
	"github.com/pajbot/pajbot2-discord/internal/mute"
	"github.com/pajbot/pajbot2-discord/pkg/utils"
)

func init() {
	cmd := &SlashCommand{
		name: "focus",

		command: &discordgo.ApplicationCommand{
			Description: "🧘 Mute yourself to enter peace mode. Silence distractions, clear your mind.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "duration",
					Description: "Mute duration",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{
							Name:  "1 Hour",
							Value: "1h",
						},
						{
							Name:  "2 Hours",
							Value: "2h",
						},
						{
							Name:  "4 Hours",
							Value: "4h",
						},
						{
							Name:  "8 Hours",
							Value: "8h",
						},
						{
							Name:  "16 Hours",
							Value: "16h",
						},
						{
							Name:  "24 Hours",
							Value: "1d",
						},
					},
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

			muted, err := mute.IsUserMuted(sqlClient, i.GuildID, executingUser.ID)
			if err != nil {
				fmt.Println("Error checking user mute:", err)

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "mute check failed",
					},
				})
				return
			}

			if muted {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "You are already self-muted dummy, no takesies backsies",
					},
				})
				return
			}

			options := i.ApplicationCommandData().Options
			muteDuration := options[1].StringValue()

			if _, err := mute.MuteUser(sqlClient, s, i.GuildID, s.State.User, executingUser, muteDuration, mute.SelfMuteReason); err != nil {
				fmt.Println("Error executing self-mute:", err)
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "an error happened running the self-mute: " + err.Error(),
					},
				})
			} else {
				fmt.Println("Self-Mute success")
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Inner peace obtained for " + muteDuration,
					},
				})

				const resultFormat = "%s self-muted for %s"
				message := fmt.Sprintf(resultFormat, utils.MentionUser(s, i.GuildID, executingUser), muteDuration)

				targetChannel := channels.Get(i.GuildID, "action-log")
				if targetChannel != "" {
					// Announce self-mute in action-log channel
					s.ChannelMessageSend(targetChannel, message)
				}
			}
		},
	}

	register(cmd)
}
