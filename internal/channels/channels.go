package channels

import (
	"database/sql"
	"fmt"
	"iter"
	"maps"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/pajbot2-discord/internal/serverconfig"
)

type ChannelData struct {
	Name        string
	DisplayName string
	Description string
}

var (
	validChannels = map[string]*ChannelData{
		"action-log": {
			Name:        "action-log",
			DisplayName: "Action log",
			Description: "",
		},
		"moderation-action": {
			Name:        "moderation-action",
			DisplayName: "Moderation action",
			Description: "",
		},
		"weeb-channel": {
			Name:        "weeb-channel",
			DisplayName: "Weeb channel",
			Description: "",
		},
		"system-messages": {
			Name:        "system-messages",
			DisplayName: "System messages",
			Description: "",
		},
		"connection-updates": {
			Name:        "connection-updates",
			DisplayName: "Connection updates",
			Description: "",
		},
		"manual-verification": {
			Name:        "manual-verification",
			DisplayName: "Manual verification",
			Description: "",
		},
		"stream-announce": {
			Name:        "stream-announce",
			DisplayName: "Stream announce",
			Description: "",
		},
	}
)

func List() iter.Seq[*ChannelData] {
	return maps.Values(validChannels)
}

func Choices() []*discordgo.ApplicationCommandOptionChoice {
	choices := []*discordgo.ApplicationCommandOptionChoice{}

	for channelKey, channelData := range validChannels {
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  channelData.DisplayName,
			Value: channelKey,
		})
	}

	return choices
}

func init() {
	if len(validChannels) > 24 {
		panic("Too many channels, things will break")
	}
}

func ValidRole(channelRole string) (ok bool) {
	_, ok = validChannels[channelRole]
	return
}

func get(guildID, channelType string) string {
	if !ValidRole(channelType) {
		fmt.Println("Invalid channel type passed to channels.get:", channelType)
		return ""
	}
	key := "channel:" + channelType
	return serverconfig.Get(guildID, key)
}

// Get returns the channel ID of the given channel type, if configured.
// If not configured, it will continue
func Get(guildID, channelType string, fallbackChannelTypes ...string) string {
	v := get(guildID, channelType)
	if v != "" {
		return v
	}

	for _, fallbackChannelType := range fallbackChannelTypes {
		if v := get(guildID, fallbackChannelType); v != "" {
			return v
		}
	}

	return ""
}

func Set(sqlClient *sql.DB, guildID, botChannel, discordChannelID string) error {
	if !ValidRole(botChannel) {
		return fmt.Errorf("invalid bot channel: %s", botChannel)
	}

	if discordChannelID == "" {
		return fmt.Errorf("missing discord channel ID")
	}

	key := fmt.Sprintf("channel:%s", botChannel)

	return serverconfig.Save(sqlClient, guildID, key, discordChannelID)
}

func Clear(sqlClient *sql.DB, guildID, botChannel string) error {
	if !ValidRole(botChannel) {
		return fmt.Errorf("invalid bot channel: %s", botChannel)
	}

	key := fmt.Sprintf("channel:%s", botChannel)

	return serverconfig.Remove(sqlClient, guildID, key)
}
