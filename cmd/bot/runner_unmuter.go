package main

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/pajbot2-discord/internal/mute"
	"github.com/pajbot/pajbot2-discord/internal/serverconfig"
	"github.com/pajbot/pajbot2-discord/pkg/utils"
)

func startUnmuterRunner(ctx context.Context, bot *discordgo.Session) {
	const resultFormat = "%s was unmuted (reason was %s)"
	const interval = 3 * time.Second

	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			unmutedUsers, err := mute.ExpireMutes(bot, sqlClient)
			if err != nil {
				fmt.Println("err:", err)
			}

			// Report unmutes in moderation-actions channel
			for _, unmutedUser := range unmutedUsers {
				member, err := bot.GuildMember(unmutedUser.GuildID, unmutedUser.UserID)
				if err != nil {
					fmt.Println("Error getting guild member:", err)
					continue
				}
				resultMessage := fmt.Sprintf(resultFormat, utils.MentionMember(member), utils.EscapeMarkdown(unmutedUser.Reason))
				targetChannel := serverconfig.Get(unmutedUser.GuildID, "channel:moderation-action")
				if targetChannel == "" {
					fmt.Println("No channel set up for moderation actions")
					break
				}
				bot.ChannelMessageSend(targetChannel, resultMessage)
			}
		}
	}

}
