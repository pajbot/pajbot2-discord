package slashcommands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/pajbot2-discord/internal/channels"
	"github.com/pajbot/pajbot2-discord/internal/mute"
	"github.com/pajbot/pajbot2-discord/pkg/utils"
)

func init() {
	var perms int64 = discordgo.PermissionBanMembers
	cmd := &SlashCommand{
		name: "mute",

		command: &discordgo.ApplicationCommand{
			Description: "Mute user",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "User to mute",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "duration",
					Description: "Mute duration",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "reason",
					Description: "Mute reason",
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
			userToMute := options[0].UserValue(s)
			muteDuration := options[1].StringValue()
			muteReason := options[2].StringValue()

			if userToMute == nil {
				fmt.Println("Invalid user to mute")
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "an invalid user was passed to the mute command?",
					},
				})
				return
			}

			if message, duration, err := mute.MuteUser(sqlClient, s, i.GuildID, moderator, userToMute, muteDuration, muteReason); err != nil {
				fmt.Println("Error executing mute:", err)
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "an error happened running the mute: " + err.Error(),
					},
				})
			} else {
				fmt.Println("Mute success")
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: message,
					},
				})

				const resultFormat = "%s muted %s for %s. reason: %s"
				message = fmt.Sprintf(resultFormat, utils.MentionUser(s, i.GuildID, moderator), utils.MentionUser(s, i.GuildID, userToMute), duration, muteReason)

				targetChannel := channels.Get(i.GuildID, "moderation-action")
				if targetChannel != "" {
					// Announce mute in moderation-action channel
					s.ChannelMessageSend(targetChannel, message)
				}
			}
		},
	}

	register(cmd)
}
