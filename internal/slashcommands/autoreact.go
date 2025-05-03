package slashcommands

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/pajbot2-discord/internal/serverconfig"
)

func init() {
	var perms int64 = discordgo.PermissionAdministrator
	cmd := &SlashCommand{
		name: "autoreact",
		command: &discordgo.ApplicationCommand{
			Description: "Configure autoreact",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "get",
					Description: "Get autoreact in a channel",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "channel",
							Description: "Channel",
							Type:        discordgo.ApplicationCommandOptionChannel,
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "set",
					Description: "Set autoreact in a channel",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "channel",
							Description: "Channel",
							Type:        discordgo.ApplicationCommandOptionChannel,
							Required:    true,
						},
						{
							Name:        "emojis",
							Description: "List of emojis to autoreact to messages in the channel. Separated by space",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    false,
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
					autoreactGet(s, i, subcommand.Options)
				case "set":
					autoreactSet(s, i, subcommand.Options)
				}
			}
		},
		registeredCommands: []*discordgo.ApplicationCommand{},
	}

	register(cmd)
}

func discordifyEmojis(emojis []string) (discordEmojis []string) {
	for _, emoji := range emojis {
		if strings.Contains(emoji, ":") {
			discordEmojis = append(discordEmojis, fmt.Sprintf("<:%s>", emoji))
		} else {
			// this is an emoji
			discordEmojis = append(discordEmojis, emoji)
		}
	}

	return
}

func autoreactGet(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	channel := options[0].ChannelValue(s)

	emojis := serverconfig.GetAutoReact(i.GuildID, channel.ID)

	if len(emojis) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Title:   "AutoReact Get",
				Content: fmt.Sprintf("No autoreact emojis set up in <#%s>", channel.ID),
			},
		})
	} else {
		autoreactEmojis := discordifyEmojis(emojis)

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Title:   "AutoReact Get",
				Content: fmt.Sprintf("Autoreact emojis set up in <#%s>: %s", channel.ID, strings.Join(autoreactEmojis, ", ")),
			},
		})
	}
}

var discordEmojiRegex = regexp.MustCompile(`<(a)?:([^<>:]+):([0-9]+)>`)

var unicodeEmojiRegex = regexp.MustCompile(`[\x{00A0}-\x{1F9EF}]`)

func autoreactSet(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	channel := options[0].ChannelValue(s)
	if len(options) == 2 {
		// set
		emojiString := options[1].StringValue()
		sanitizedEmojis := []string{}

		for _, emoji := range strings.Split(emojiString, " ") {
			v := strings.TrimSpace(emoji)
			if v == "" {
				continue
			}

			if unicodeEmojiRegex.MatchString(v) {
				// just an emoji
				sanitizedEmojis = append(sanitizedEmojis, v)
			} else {
				discordEmoji := discordEmojiRegex.FindStringSubmatch(v)
				if discordEmoji == nil {
					fmt.Println("bad emoji:", v)
				} else {
					sanitizedEmojis = append(sanitizedEmojis, fmt.Sprintf("%s:%s", discordEmoji[2], discordEmoji[3]))
				}
			}
		}

		err := serverconfig.SetAutoReact(sqlClient, i.GuildID, channel.ID, sanitizedEmojis)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Title:   "AutoReact Set",
					Content: fmt.Sprintf("Error setting autoreact in <#%s>: %s", channel.ID, err),
				},
			})
		} else {
			autoreactEmojis := discordifyEmojis(sanitizedEmojis)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Title:   "AutoReact Set",
					Content: fmt.Sprintf("Successfully updated autoreact emojis in <#%s> to %s", channel.ID, strings.Join(autoreactEmojis, ", ")),
				},
			})
		}
	} else {
		// reset
		err := serverconfig.RemoveAutoReact(sqlClient, i.GuildID, channel.ID)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Title:   "AutoReact Set (remove)",
					Content: fmt.Sprintf("Error resetting autoreact in <#%s>: %s", channel.ID, err),
				},
			})
		} else {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Title:   "AutoReact Set (remove)",
					Content: fmt.Sprintf("Successfully reset autoreact emojis in <#%s>", channel.ID),
				},
			})
		}
	}
}
