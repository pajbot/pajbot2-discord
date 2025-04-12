package slashcommands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/pajbot2-discord/internal/channels"
)

func init() {
	var perms int64 = discordgo.PermissionAdministrator
	cmd := &SlashCommand{
		name: "channel",
		command: &discordgo.ApplicationCommand{
			Description: "Configure channel",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "clear",
					Description: "Clear channel",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "bot-channel",
							Description: "Bot channel",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
							Choices:     channels.Choices(),
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "get",
					Description: "Get information about a specific channel",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "bot-channel",
							Description: "Bot channel",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
							Choices:     channels.Choices(),
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "set",
					Description: "Set channel",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "bot-channel",
							Description: "Bot channel",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
							Choices:     channels.Choices(),
						},
						{
							Name:        "discord-channel",
							Description: "Discord channel",
							Type:        discordgo.ApplicationCommandOptionChannel,
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "sethere",
					Description: "Set channel here",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "bot-channel",
							Description: "Bot channel",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
							Choices:     channels.Choices(),
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "list",
					Description: "List channels",
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
				case "set":
					channelSet(s, i, subcommand.Options)
				case "sethere":
					channelSetHere(s, i, subcommand.Options)
				case "get":
					channelGet(s, i, subcommand.Options)
				case "clear":
					channelClear(s, i, subcommand.Options)
				case "list":
					channelList(s, i)
				}
			}
		},
		registeredCommands: []*discordgo.ApplicationCommand{},
	}

	register(cmd)
}

func channelSet(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	botChannel := options[0].StringValue()
	discordChannelID := options[1].ChannelValue(s).ID

	if err := channels.Set(sqlClient, i.GuildID, botChannel, discordChannelID); err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Error setting %s channel to <#%s>: %s", botChannel, discordChannelID, err),
			},
		})
	} else {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Set channel %s to <#%s>", botChannel, discordChannelID),
			},
		})
	}
}

func channelSetHere(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	if i.Interaction.ChannelID == "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "set here only works in a real channel",
			},
		})
		return
	}

	botChannel := options[0].StringValue()
	discordChannelID := i.Interaction.ChannelID

	if err := channels.Set(sqlClient, i.GuildID, botChannel, discordChannelID); err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Error setting %s channel to <#%s>: %s", botChannel, discordChannelID, err),
			},
		})
	} else {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Set channel %s to <#%s>", botChannel, discordChannelID),
			},
		})
	}
}

func formatChannel(botChannel, discordChannelID string) string {
	if discordChannelID == "" {
		return fmt.Sprintf("%s: not configured", botChannel)
	}

	return fmt.Sprintf("%s: <#%s>", botChannel, discordChannelID)
}

func channelGet(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	botChannel := options[0].StringValue()

	discordChannelID := channels.Get(i.GuildID, botChannel)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Channel info: %s", formatChannel(botChannel, discordChannelID)),
		},
	})
}

func channelClear(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	botChannel := options[0].StringValue()

	if err := channels.Clear(sqlClient, i.GuildID, botChannel); err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Error clearing %s channel: %s", botChannel, err),
			},
		})
	} else {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Cleared channel %s", botChannel),
			},
		})
	}
}

func channelList(s *discordgo.Session, i *discordgo.InteractionCreate) {
	response := ""
	first := true

	for channel := range channels.List() {
		if !first {
			response += "\n"
		}

		discordChannelID := channels.Get(i.GuildID, channel.Name)

		response += formatChannel(channel.Name, discordChannelID)

		first = false
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: response,
		},
	})
}
