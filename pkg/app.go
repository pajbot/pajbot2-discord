package pkg

import "github.com/pajbot/pajbot2-discord/internal/connection"

type App interface {
	RegisterUser(pendingConnection *connection.PendingConnection, twitchUser connection.TwitchValidateResponse)
}
