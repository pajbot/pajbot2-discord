package twitchstreamannouncer

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/nicklaw5/helix"
	"github.com/pajbot/pajbot2-discord/internal/channels"
	"github.com/pajbot/pajbot2-discord/internal/serverconfig"
)

func makeMessage(s *helix.Stream) string {
	if s == nil {
		return "bad code"
	}

	return fmt.Sprintf("%s is now live! https://twitch.tv/%s", s.UserName, s.UserLogin)
}

func Start(ctx context.Context, guildID string, helixClient *helix.Client, s *discordgo.Session, sqlClient *sql.DB) {
	fmt.Println(guildID, "twitchstreamannouncer start")

	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ctx.Done():
			fmt.Println("exiting twitchstreamannouncer")
			return

		case <-ticker.C:
			streamAnnounceChannel := channels.Get(guildID, "stream-announce")
			if streamAnnounceChannel == "" {
				continue
			}

			fmt.Println(guildID, "twitchstreamannouncer tick")

			nowOnlineStreamers, err := checkForStreamChanges(guildID, helixClient, sqlClient)
			if err != nil {
				fmt.Println("Error checking live status of streamers:", err)
			} else {
				if len(nowOnlineStreamers) > 0 {
					for _, stream := range nowOnlineStreamers {
						fmt.Printf("Send stream announce for %#v\n", stream)
						_, err := s.ChannelMessageSend(streamAnnounceChannel, makeMessage(stream))
						if err != nil {
							// TODO: should we kill the sql entry if the message sending fails?
							fmt.Println("Error sending stream announcement", err)
						}
					}
				}
			}
		}
	}
}

func checkForStreamChanges(guildID string, helixClient *helix.Client, sqlClient *sql.DB) (nowOnlineStreamers []*helix.Stream, err error) {
	streamIDs := strings.Split(serverconfig.GetValue(guildID, "stream_ids"), ",")

	params := helix.StreamsParams{
		UserIDs: streamIDs,
	}

	var response *helix.StreamsResponse

	fmt.Println(guildID, "Check for stream changes", streamIDs)

	response, err = helixClient.GetStreams(&params)
	if err != nil {
		return
	}

	for _, twitchUserID := range streamIDs {
		fmt.Println(guildID, "Seeing if stream was returned for", twitchUserID)
		isLive := false
		var matchingStream *helix.Stream
		for _, stream := range response.Data.Streams {
			if stream.UserID == twitchUserID {
				fmt.Println(guildID, "Stream did exist", twitchUserID)
				isLive = true
				matchingStream = &stream
				break
			}
		}

		if isLive {
			const query = `INSERT INTO twitchstreamannouncer (twitch_user_id, twitch_stream_id, discord_guild_id) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING;`
			res, err := sqlClient.Exec(query, matchingStream.UserID, matchingStream.ID, guildID)
			if err != nil {
				return nil, err
			}

			rowsAffected, err := res.RowsAffected()
			if err != nil {
				return nil, err
			}
			if rowsAffected == 1 {
				fmt.Printf("[%s] Stream %s has come online\n", guildID, twitchUserID)
				nowOnlineStreamers = append(nowOnlineStreamers, matchingStream)
			}
		} else {
			const query = `DELETE FROM twitchstreamannouncer WHERE twitch_user_id=$1 AND discord_guild_id=$2;`
			res, err := sqlClient.Exec(query, twitchUserID, guildID)
			if err != nil {
				return nil, err
			}

			rowsAffected, err := res.RowsAffected()
			if err != nil {
				return nil, err
			}
			if rowsAffected == 1 {
				fmt.Printf("[%s] Stream %s has gone offline\n", guildID, twitchUserID)
			}
		}
	}

	return
}
