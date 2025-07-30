package twitchstreamannouncer

import (
	"context"
	"database/sql"
	"fmt"
	"iter"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/nicklaw5/helix"
	"github.com/pajbot/pajbot2-discord/internal/channels"
	"github.com/pajbot/pajbot2-discord/internal/serverconfig"
)

var (
	guilds      = map[string]struct{}{}
	guildsMutex = &sync.Mutex{}
)

func makeMessage(s *helix.Stream) string {
	if s == nil {
		return "bad code"
	}

	return fmt.Sprintf("%s is now live! https://twitch.tv/%s", s.UserName, s.UserLogin)
}

func Register(guildID string) {
	guildsMutex.Lock()
	defer guildsMutex.Unlock()
	fmt.Printf("[TSA] Register %s\n", guildID)
	guilds[guildID] = struct{}{}
}

func getGuildIDs() iter.Seq[string] {
	guildsMutex.Lock()
	defer guildsMutex.Unlock()
	return maps.Keys(guilds)
}

type guildInfo struct {
	guildID               string
	streamAnnounceChannel string
}

// Accepts a list of guild IDs, and returns a map with the Twitch User ID mapping to a list of Guild IDs interested in that streamer
func getGuildToStreamerMap(guildIDs iter.Seq[string]) map[string][]guildInfo {
	streamerMap := map[string][]guildInfo{}

	for guildID := range guildIDs {
		streamAnnounceChannel := channels.Get(guildID, "stream-announce")
		if streamAnnounceChannel == "" {
			continue
		}

		// Streams that this guild is interested in
		streamIDs := strings.Split(serverconfig.GetValue(guildID, "stream_ids"), ",")
		if len(streamIDs) == 0 {
			// Guild is not interested in any Twitch channels
			continue
		}

		for _, streamID := range streamIDs {
			streamerMap[streamID] = append(streamerMap[streamID], guildInfo{
				guildID:               guildID,
				streamAnnounceChannel: streamAnnounceChannel,
			})
		}
	}

	return streamerMap
}

func tick(sqlClient *sql.DB, s *discordgo.Session, helixClient *helix.Client) error {
	guildIDs := getGuildIDs()
	streamerMap := getGuildToStreamerMap(guildIDs)
	streamsToCheck := maps.Keys(streamerMap)

	fmt.Printf("[TSA] Polling for stream changes (%#v)\n", streamerMap)

	streams, err := checkForStreamChanges(helixClient, streamsToCheck)
	if err != nil {
		return err
	}

	for twitchUserID, interestedGuilds := range streamerMap {
		isLive := false
		var matchingStream *helix.Stream
		for _, stream := range streams {
			if stream.UserID == twitchUserID {
				isLive = true
				matchingStream = &stream
				break
			}
		}

		for _, guild := range interestedGuilds {
			guildID := guild.guildID
			streamAnnounceChannel := guild.streamAnnounceChannel

			if isLive {
				if matchingStream == nil {
					panic("matching stream must not be nil")
				}

				const query = `INSERT INTO twitchstreamannouncer (twitch_user_id, twitch_stream_id, discord_guild_id) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING;`
				res, err := sqlClient.Exec(query, matchingStream.UserID, matchingStream.ID, guildID)
				if err != nil {
					return fmt.Errorf("failed to insert live status for %s in guild %s: %w", twitchUserID, guildID, err)
				}

				rowsAffected, err := res.RowsAffected()
				if err != nil {
					return fmt.Errorf("failed to get live status rows affected for %s in guild %s: %w", twitchUserID, guildID, err)
				}

				if rowsAffected == 1 {
					fmt.Printf("[TSA] Stream %s has come online (guild %s)\n", guildID, twitchUserID)
					_, err := s.ChannelMessageSend(streamAnnounceChannel, makeMessage(matchingStream))
					if err != nil {
						// TODO: should we kill the sql entry if the message sending fails?
						fmt.Printf("[TSA] Error sending stream announcement for %s in guild %s: %s\n", twitchUserID, guildID, err)
					}
				}
			} else {
				const query = `DELETE FROM twitchstreamannouncer WHERE twitch_user_id=$1 AND discord_guild_id=$2;`
				res, err := sqlClient.Exec(query, twitchUserID, guildID)
				if err != nil {
					return fmt.Errorf("failed to delete live status for %s in guild %s: %w", twitchUserID, guildID, err)
				}

				rowsAffected, err := res.RowsAffected()
				if err != nil {
					return fmt.Errorf("failed to get offline status rows affected for %s in guild %s: %w", twitchUserID, guildID, err)
				}
				if rowsAffected == 1 {
					fmt.Printf("[TSA] Stream %s has gone offline (guild %s)\n", twitchUserID, guildID)
				}
			}
		}
	}

	return nil
}

func Start(ctx context.Context, helixClient *helix.Client, s *discordgo.Session, sqlClient *sql.DB) {
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ctx.Done():
			fmt.Println("exiting twitchstreamannouncer")
			return

		case <-ticker.C:
			if err := tick(sqlClient, s, helixClient); err != nil {
				fmt.Printf("[TSA] Error in tick: %s\n", err)
			}
		}
	}
}

func checkForStreamChanges(helixClient *helix.Client, streamIDsSeq iter.Seq[string]) ([]helix.Stream, error) {
	streamIDs := slices.Collect(streamIDsSeq)
	params := helix.StreamsParams{
		UserIDs: streamIDs,
	}

	var response *helix.StreamsResponse

	response, err := helixClient.GetStreams(&params)
	if err != nil {
		return nil, err
	}

	return response.Data.Streams, err
}
