package utils

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

const (
	maxMessageSize = 2000
)

// SendChunks will take your list of chunks, pair them together, and send them to the given channel surrounded by your given prefix and suffix
// If your chunks paired together don't fit in a single discord message, it will split them up, ensuring both messages that are sent are surrounded by the prefix and suffix
func SendChunks(prefix, suffix string, chunks []string, channelID string, s *discordgo.Session) {
	var output string
	messages := []string{}
	psLength := len(prefix) + len(suffix)
	for _, chunk := range chunks {
		if len(chunk)+psLength >= maxMessageSize {
			fmt.Println("Single chunk is too long")
			return
		}

		if len(output)+len(chunk)+psLength >= maxMessageSize {
			// chunk it
			messages = append(messages, prefix+output+suffix)
			output = ""
		}
		output += chunk
	}

	if len(output) > 0 {
		// add remainder, if it exists, to messages
		messages = append(messages, prefix+output+suffix)
	}

	for _, message := range messages {
		_, err := s.ChannelMessageSend(channelID, message)
		if err != nil {
			fmt.Println("Error in chunk send:", err)
		}
	}
}
