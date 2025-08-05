package slashcommands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/pajbot2-discord/internal/commands/mute"
)

func init() {
	cmd := &SlashCommand{
		name: "selfmute",

		command: &discordgo.ApplicationCommand{
			Description: "Zip your own mouth",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "duration",
					Description: "Mute duration",
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
			muteDuration := options[1].StringValue()
			muteReason := "Self Mute"

			if message, err := mute.Execute(s, i.GuildID, s.State.User, executingUser, muteDuration, muteReason); err != nil {
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
						Content: message,
					},
				})
			}
		},
	}

	register(cmd)
}
