package slashcommands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/pajbot2-discord/internal/commands/ping"
)

func init() {
	cmd := &SlashCommand{
		name: "ping",

		command: &discordgo.ApplicationCommand{
			Description: "Ping Pong",
		},

		handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: ping.GetPingResponse(),
				},
			})
		},
	}

	register(cmd)
}
