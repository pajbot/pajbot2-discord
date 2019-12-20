package autoreact

import (
	"strings"

	"github.com/pajbot/pajbot2-discord/internal/serverconfig"
)

func Get(serverID, channelID string) (emojiIDs []string) {
	key := "autoreact:" + channelID
	s := serverconfig.Get(serverID, key)
	if s == "" {
		return
	}

	return strings.Split(s, ",")
}
