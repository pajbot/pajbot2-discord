package mute

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/pajbot2-discord/internal/config"
)

type MutedUser struct {
	UserID  string
	GuildID string
	Reason  string
}

func IsUserMuted(sqlClient *sql.DB, userID string) (muted bool, err error) {
	// Check if the user is supposed to be muted
	const query = "SELECT reason FROM discord_mutes WHERE user_id=$1"
	row := sqlClient.QueryRow(query, userID)

	var reason string
	err = row.Scan(&reason)
	if err != nil {
		if err == sql.ErrNoRows {
			muted = false
			err = nil
			return
		}

		fmt.Println("Error checking users mute:", err)
		return
	}

	muted = true

	return
}

func ExpireMutes(s *discordgo.Session, sqlClient *sql.DB) (unmutedUsers []MutedUser, err error) {
	now := time.Now()
	const query = `SELECT id, guild_id, user_id, reason, mute_end FROM discord_mutes ORDER BY mute_end ASC LIMIT 30;`
	rows, err := sqlClient.Query(query)
	if err != nil {
		fmt.Println("err:", err)
		return
	}
	var rowsToRemove []int64
	defer func() {
		for _, id := range rowsToRemove {
			sqlClient.Exec("DELETE FROM discord_mutes WHERE id=$1", id)
		}
	}()
	defer rows.Close()
	for rows.Next() {
		var id int64
		var user MutedUser
		var muteEnd time.Time

		err = rows.Scan(&id, &user.GuildID, &user.UserID, &user.Reason, &muteEnd)
		if err != nil {
			return
		}

		if muteEnd.After(now) {
			return
		}

		err = s.GuildMemberRoleRemove(user.GuildID, user.UserID, config.MutedRole)
		if err != nil {
			fmt.Println("Error removing role")
			continue
		}

		unmutedUsers = append(unmutedUsers, user)

		rowsToRemove = append(rowsToRemove, id)
	}

	return
}
