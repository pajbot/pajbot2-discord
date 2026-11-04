package slashcommands

import (
	"fmt"
	"slices"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/pajbot2-discord/internal/channels"
	"github.com/pajbot/pajbot2-discord/internal/config"
	"github.com/pajbot/pajbot2-discord/internal/connection"
	"github.com/pajbot/pajbot2-discord/internal/roles"
)

func init() {
	cmd := &SlashCommand{
		name: "auth",

		command: &discordgo.ApplicationCommand{
			Description: "Authenticate user",
		},

		handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.Member == nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "no user found?",
					},
				})
				return
			}

			if slices.Contains(i.Member.Roles, roles.GetSingle(i.GuildID, "member")) {
				fmt.Println("User has members role")

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "You are already authenticated.",

						Flags: discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			existingConnection, err := connection.Get(sqlClient, i.GuildID, i.Member.User.ID)
			if err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("An error occurred while checking your existing connection: %s", err),

						Flags: discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			if existingConnection != nil {
				// User had a connection already
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("You had a pre-existing connection that we re-used %s (%s)", existingConnection.TwitchUserLogin, existingConnection.TwitchUserID),

						Flags: discordgo.MessageFlagsEphemeral,
					},
				})
				if err := roles.Grant(s, i.GuildID, i.Member.User.ID, "member"); err != nil {
					fmt.Println("Error adding Members role to user:", err)
				} else {
					fmt.Printf("Granted Members role to %s (%s)\n", i.Member.User.Username, i.Member.User.ID)
				}
				return
			}

			// Generate connection ID
			connectionID := connection.RegisterPendingConnection(i.Member.User.ID, i.GuildID)

			// Put that connection ID in a memory map
			connection.RegisterPendingConnection(connectionID, i.Member.User.ID)

			// TODO: Generate URL based on config stuff
			connectionURL := fmt.Sprintf("%s/login?code=%s", config.Domain, connectionID)

			privacyPolicy := fmt.Sprintf("%s/legal/privacy-policy", config.Domain)
			termsOfService := fmt.Sprintf("%s/legal/terms-of-service", config.Domain)

			manualVerificationChannel := channels.Get(i.GuildID, "manual-verification")
			messageContent := fmt.Sprintf("Hi! You need to authenticate to join the Discord. Please click [this link](<%s>) and sign in with your Twitch account to continue.\n\nPlease review our [privacy policy](<%s>) and [terms of service](<%s>) before proceeding.\n\nIf you run into any issues, ask for help in the <#%s> channel, or rejoin the server to try again. If the channel says **No Access** you are either already verified, or you are not in the server.", connectionURL, privacyPolicy, termsOfService, manualVerificationChannel)

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: messageContent,

					Flags: discordgo.MessageFlagsEphemeral,
				},
			})

		},
	}

	register(cmd)
}
