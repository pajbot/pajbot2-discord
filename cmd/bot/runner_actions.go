package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/pajbot2-discord/pkg"
)

func startActionRunner(ctx context.Context, bot *discordgo.Session) {
	const interval = 3 * time.Second

	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			now := time.Now()
			const query = `SELECT id, action, timepoint FROM discord_queue ORDER BY timepoint DESC LIMIT 30;`
			rows, err := sqlClient.Query(query)
			if err != nil {
				fmt.Println("err:", err)
				continue
			}
			var actionsToRemove []int64
			defer rows.Close()
			for rows.Next() {
				var (
					id           int64
					actionString string
					timepoint    time.Time
				)
				if err = rows.Scan(&id, &actionString, &timepoint); err != nil {
					fmt.Println("Error scanning:", err)
				}
				if timepoint.After(now) {
					continue
				}

				var action pkg.Action
				err = json.Unmarshal([]byte(actionString), &action)
				if err != nil {
					fmt.Println("Error unmarshaling action:", err)
					continue
				}
				fmt.Println("Perform", action.Type)

				switch action.Type {
				case "unmute":
					err = bot.GuildMemberRoleRemove(action.GuildID, action.UserID, action.RoleID)
					if err != nil {
						if rErr, ok := err.(*discordgo.RESTError); ok {
							if rErr.Message != nil {
								dErr := rErr.Message
								if dErr.Code == 10007 {
									// User not in server anymore
								} else {
									fmt.Println("1 Error removing role", err, dErr.Code)
									continue
								}
							} else {
								fmt.Println("2 Error removing role", err)
								continue
							}
						} else {
							fmt.Println("3 Error removing role", err)
							continue
						}
					}

					actionsToRemove = append(actionsToRemove, id)
				}
			}

			for _, actionID := range actionsToRemove {
				sqlClient.Exec("DELETE FROM discord_queue WHERE id=$1", actionID)
			}
		}
	}
}
