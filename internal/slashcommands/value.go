package slashcommands

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/pajbot2-discord/internal/serverconfig"
	"github.com/pajbot/pajbot2-discord/internal/values"
)

var memberRoleNames = map[string]string{
	"0": "Disabled",
	"1": "Auto grant on join",
	"2": "Required Twitch verification",
}

func init() {
	var perms int64 = discordgo.PermissionAdministrator
	cmd := &SlashCommand{
		name: "value",
		command: &discordgo.ApplicationCommand{
			Description: "Configure value",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "get",
					Description: "List values",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "twitch-announce",
					Description: "Set Twitch IDs to announce live status for",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "stream-ids",
							Description: "Comma-separated list of Twitch IDs to announce live status for",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    false,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "pajbot-host",
					Description: "Set the Pajbot host to be used for any Pajbot API requests",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "pajbot-host",
							Description: "e.g. forsen.tv",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    false,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "member-role-mode",
					Description: "Member role mode",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "member-role-mode",
							Description: "Member role mode",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
							Choices: []*discordgo.ApplicationCommandOptionChoice{
								{
									Name:  "Disabled",
									Value: "0",
								},
								{
									Name:  "Auto grant role on join",
									Value: "1",
								},
								{
									Name:  "Require Twitch verification",
									Value: "2",
								},
							},
						},
					},
				},
			},
			DefaultMemberPermissions: &perms,
		},
		handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			switch i.Type {
			case discordgo.InteractionApplicationCommand:
				options := i.ApplicationCommandData().Options
				subcommand := options[0]
				switch subcommand.Name {
				case "get":
					valueGet(s, i)
				case "twitch-announce":
					valueTwitchAnnounce(s, i, subcommand.Options)
				case "pajbot-host":
					valuePajbotHost(s, i, subcommand.Options)
				case "member-role-mode":
					valueMemberRoleMode(s, i, subcommand.Options)
				}
			}
		},
		registeredCommands: []*discordgo.ApplicationCommand{},
	}

	register(cmd)
}

func valueGet(s *discordgo.Session, i *discordgo.InteractionCreate) {
	embeds := []*discordgo.MessageEmbed{}
	embed := &discordgo.MessageEmbed{
		Type:  discordgo.EmbedTypeRich,
		Title: "Values",
	}

	embeds = append(embeds, embed)

	if streamIDs := serverconfig.GetValue(i.GuildID, values.StreamIDs); streamIDs != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Stream IDs",
			Value:  streamIDs,
			Inline: false,
		})
	} else {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Stream IDs",
			Value:  "Unset",
			Inline: false,
		})
	}

	if memberRoleMode := serverconfig.GetValue(i.GuildID, values.MemberRoleMode); memberRoleMode != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Member role mode",
			Value:  memberRoleNames[memberRoleMode],
			Inline: false,
		})
	} else {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Member role mode",
			Value:  "Unset",
			Inline: false,
		})
	}

	if pajbotHost := serverconfig.GetValue(i.GuildID, values.PajbotHost); pajbotHost != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Pajbot host",
			Value:  pajbotHost,
			Inline: false,
		})
	} else {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Pajbot host",
			Value:  "Unset",
			Inline: false,
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Title:  "Values",
			Embeds: embeds,
		},
	})
}

func valueTwitchAnnounce(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	if len(options) == 0 {
		if err := serverconfig.RemoveValue(sqlClient, i.GuildID, values.StreamIDs); err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("Error: %s", err),
				},
			})
		} else {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Removed",
				},
			})
		}
		return
	}

	twitchUserIDs := strings.Split(options[0].StringValue(), ",")

	sanitizedValue := []string{}

	for _, rawValue := range twitchUserIDs {
		v := strings.TrimSpace(rawValue)
		sanitizedValue = append(sanitizedValue, v)
	}

	if err := serverconfig.SetValue(sqlClient, i.GuildID, values.StreamIDs, strings.Join(sanitizedValue, ",")); err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Error: %s", err),
			},
		})
	} else {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Updated to %v", sanitizedValue),
			},
		})
	}
}

func valuePajbotHost(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	if len(options) == 0 {
		if err := serverconfig.RemoveValue(sqlClient, i.GuildID, values.PajbotHost); err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("Error: %s", err),
				},
			})
		} else {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Removed",
				},
			})
		}
		return
	}

	pajbotHost := options[0].StringValue()

	if err := serverconfig.SetValue(sqlClient, i.GuildID, values.PajbotHost, pajbotHost); err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Error: %s", err),
			},
		})
	} else {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Updated to %s", pajbotHost),
			},
		})
	}
}

func valueMemberRoleMode(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	modeValue := options[0].StringValue()
	modeName := memberRoleNames[modeValue]

	fmt.Println("Setting member role mode to", modeValue, modeName)

	if err := serverconfig.SetValue(sqlClient, i.GuildID, values.MemberRoleMode, modeValue); err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Error: %s", err),
			},
		})
	} else {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Updated to %s", modeName),
			},
		})
	}
}
