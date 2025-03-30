package slashcommands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/pajbot2-discord/internal/commands/mute"
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
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "an invalid user was passed to the mute command?",
					},
				})
				return
			}

			options := i.ApplicationCommandData().Options
			userToMute := options[0].UserValue(s)
			muteDuration := options[1].StringValue()
			muteReason := options[2].StringValue()

			if userToMute == nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "an invalid user was passed to the mute command?",
					},
				})
				return
			}

			if message, err := mute.Execute(s, i.GuildID, moderator, userToMute, muteDuration, muteReason); err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "an error happened running the mute: " + err.Error(),
					},
				})
			} else {
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
