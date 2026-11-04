package format

import (
	"fmt"
	"strings"

	"github.com/pajbot/pajbot2-discord/internal/connection"
	"github.com/pajbot/pajbot2-discord/pkg/utils"
)

func Connection(connection *connection.DiscordConnection) string {
	if connection == nil {
		panic("connection may not be nil in formatConnection")
	}

	return fmt.Sprintf("`%s` (TUID `%s`)", utils.EscapeCodeBlock(connection.TwitchUserLogin), connection.TwitchUserID)
}

func AltConnections(altConnections []*connection.DiscordConnection) string {
	var b strings.Builder

	first := true

	for _, c := range altConnections {
		if !first {
			b.WriteString(", ")
		}

		fmt.Fprintf(&b, "did: `%s`, dname: `%s`", c.DiscordUserID, utils.EscapeCodeBlock(c.DiscordUserName))

		first = false
	}

	return b.String()
}
