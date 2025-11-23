package slashcommands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/pajbot2-discord/internal/channelrole"
)

func init() {
	var perms int64 = discordgo.PermissionManageRoles
	cmd := &SlashCommand{
		name: "createchannelrole",

		command: &discordgo.ApplicationCommand{
			Description: "Creates a new channel role",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "name",
					Description: "Name of the new channel role",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionChannel,
					Name:        "channel",
					Description: "Channel to bind the role to",
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
			name := options[1].StringValue()
			channel := options[2].ChannelValue(s)
			channelID := channel.ID

			if name == "" {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:   discordgo.MessageFlagsEphemeral,
						Content: "an invalid name was passed to the create channel role command",
					},
				})
				return
			}

			if channelID == "" {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:   discordgo.MessageFlagsEphemeral,
						Content: "an invalid channel was passed to the create channel role command",
					},
				})
				return
			}

			role, err := s.GuildRoleCreate(i.GuildID, &discordgo.RoleParams{
				Name: name,
			})

			if err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:   discordgo.MessageFlagsEphemeral,
						Content: "an invalid channel was passed to the create channel role command",
					},
				})
				return
			}

			err = channelrole.Create(s, moderator, i.GuildID, channelID, role.ID, role.Name)
			if err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:   discordgo.MessageFlagsEphemeral,
						Content: "error creating channel role:" + err.Error(),
					},
				})
				return
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags:   discordgo.MessageFlagsEphemeral,
					Content: fmt.Sprintf("Channel role %s created and bound to %s", name, channelID),
				},
			})
		},
	}

	register(cmd)
}
