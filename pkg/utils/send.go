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

// SplitIntoChunks splits the data into chunks of chunkSize size
func SplitIntoChunks(chunkSize int, data []string) (chunks [][]string) {
	for len(data) > 0 {
		if len(data) >= chunkSize {
			chunks = append(chunks, data[:chunkSize])
			data = data[chunkSize:]
		} else {
			chunks = append(chunks, data)
			data = []string{}
		}
	}
	return
}

// DeleteChunks will take your list of message IDs, pair them in chunks of 100, and bulk delete them
func DeleteChunks(s *discordgo.Session, channelID string, messageIDs []string) (err error) {
	messageIDChunks := SplitIntoChunks(100, messageIDs)

	for _, chunk := range messageIDChunks {
		if len(chunk) > 1 {
			err = s.ChannelMessagesBulkDelete(channelID, chunk)
			if err != nil {
				break
			}
		} else {
			err = s.ChannelMessageDelete(channelID, chunk[0])
			if err != nil {
				break
			}
		}
	}

	return
}
