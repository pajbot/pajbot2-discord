package serverconfig

import (
	"database/sql"
	"fmt"
	"strings"
)

// GetAutoReact returns a list of emoji IDs for the given channel
func GetAutoReact(guildID, channelID string) []string {
	key := "autoreact:" + channelID

	v := Get(guildID, key)

	if v == "" {
		return []string{}
	}

	return strings.Split(v, ",")
}

// SetAutoReact sets a list of emoji IDs for a given channel
func SetAutoReact(sqlClient *sql.DB, guildID, channelID string, emojis []string) error {
	key := "autoreact:" + channelID

	if len(emojis) == 0 {
		return fmt.Errorf("must set at least 1 emoji")
	}

	cleanedEmojis := make([]string, len(emojis))

	for i, emoji := range emojis {
		cleanedEmoji := strings.TrimSpace(emoji)
		if cleanedEmoji == "" {
			return fmt.Errorf("emoji in autoreact must not be empty")
		}
		cleanedEmojis[i] = cleanedEmoji
	}

	v := strings.Join(cleanedEmojis, ",")

	return Save(sqlClient, guildID, key, v)
}

func RemoveAutoReact(sqlClient *sql.DB, guildID, channelID string) error {
	key := "autoreact:" + channelID
	return Remove(sqlClient, guildID, key)
}
