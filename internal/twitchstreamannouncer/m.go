package twitchstreamannouncer

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/nicklaw5/helix"
	"github.com/pajbot/pajbot2-discord/internal/channels"
	"github.com/pajbot/pajbot2-discord/internal/serverconfig"
)

var (
	// Twitch User ID -> Live Status (true = online, false = offline)
	streams = map[string]*streamState{}

	streamsMutex = sync.Mutex{}
)

type streamer struct {
	Name string
}

type streamState struct {
	Live            bool
	UserLogin       string
	UserDisplayName string
	Game            string
	Title           string
}

func makeMessage(s *streamState) string {
	if s == nil {
		return "bad code"
	}

	if s.Live {
		return fmt.Sprintf("%s is now live! https://twitch.tv/%s", s.UserDisplayName, s.UserLogin)
	} else {
		return fmt.Sprintf("%s has gone offline! https://twitch.tv/%s", s.UserDisplayName, s.UserLogin)
	}
}

func Start(ctx context.Context, guildID string, helixClient *helix.Client, s *discordgo.Session) {
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

			nowOfflineStreamers, nowOnlineStreamers, err := checkForStreamChanges(guildID, helixClient)
			if err != nil {
				fmt.Println("Error checking live status of streamers:", err)
			} else {
				if len(nowOnlineStreamers) > 0 {
					for _, stream := range nowOnlineStreamers {
						fmt.Printf("Send stream announce for %#v\n", stream)
						_, err := s.ChannelMessageSend(streamAnnounceChannel, makeMessage(&stream))
						if err != nil {
							fmt.Println("Error sending stream announcement", err)
						}
					}
				}
				if len(nowOfflineStreamers) > 0 {
					for _, stream := range nowOnlineStreamers {
						fmt.Printf("Stream has gone offline %#v\n", stream)
						// Currently don't announce, maybe option if someone asks xd
					}
				}
			}
		}
	}
}

func checkForStreamChanges(guildID string, helixClient *helix.Client) (nowOfflineStreamers []streamState, nowOnlineStreamers []streamState, err error) {
	streamIDs := strings.Split(serverconfig.GetValue(guildID, "stream_ids"), ",")

	// fmt.Println("Checking stream status of", streamIDs)

	params := helix.StreamsParams{
		UserIDs: streamIDs,
	}

	var response *helix.StreamsResponse

	response, err = helixClient.GetStreams(&params)
	if err != nil {
		return
	}

	streamsMutex.Lock()
	defer streamsMutex.Unlock()

	for _, twitchUserID := range streamIDs {
		isLive := false
		var matchingStream *helix.Stream
		for _, stream := range response.Data.Streams {
			if stream.UserID == twitchUserID {
				isLive = true
				matchingStream = &stream
				break
			}
		}

		prevState := streams[twitchUserID]
		if prevState == nil {
			// No previous state, never notify.
			// This means we _could_ technically miss a going live notification if we have to restart bot right before a streamer goes live, but xD
			prevState = &streamState{
				Live:            false, // TODO: replace with isLive probably
				UserLogin:       "",
				UserDisplayName: "",
				Game:            "",
				Title:           "",
			}

		}

		if matchingStream != nil {
			prevState.Game = matchingStream.GameName
			prevState.Title = matchingStream.Title
			prevState.UserLogin = matchingStream.UserLogin
			prevState.UserDisplayName = matchingStream.UserName
		}

		if !prevState.Live && isLive {
			prevState.Live = true
			// Stream just went live!
			// fmt.Println("STREAM WENT LIVE", matchingStream)
			nowOnlineStreamers = append(nowOnlineStreamers, *prevState)
		} else if prevState.Live && !isLive {
			prevState.Live = false
			// Stream just went offline
			// fmt.Println("STREAM WENT OFFLINE", matchingStream)
			nowOfflineStreamers = append(nowOfflineStreamers, *prevState)
		}

		streams[twitchUserID] = prevState
	}

	return
}
