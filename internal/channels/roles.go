package channels

import (
	"fmt"

	"github.com/pajbot/pajbot2-discord/internal/serverconfig"
)

var (
	validChannelRoles = map[string]bool{
		"action-log":         true,
		"moderation-action":  true,
		"weeb-channel":       true,
		"system-messages":    true,
		"connection-updates": true,
	}
)

func ValidRole(channelRole string) (ok bool) {
	_, ok = validChannelRoles[channelRole]
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
