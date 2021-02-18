package mute

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/pajbot2-discord/internal/roles"
)

// MutedUser describes a user that is/was muted
type MutedUser struct {
	UserID  string
	GuildID string
	Reason  string
}

// IsUserMuted check if there's a mute active for the given user in the given server
func IsUserMuted(sqlClient *sql.DB, guildID, userID string) (muted bool, err error) {
	// Check if the user is supposed to be muted
	const query = "SELECT reason FROM discord_mutes WHERE guild_id=$1 AND user_id=$2"
	row := sqlClient.QueryRow(query, guildID, userID)

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

// ExpireMutes polls the database for any mutes that may have ended recently.
// Any users who should be unmuted will have their "Muted" role removed, and the muted entry will be removed from the database.
// The `unmutedUsers` return value indicates what users were unmuted, what their mute reason was, and what server they were muted in.
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

		mutedRole := roles.GetSingle(user.GuildID, "muted")
		if mutedRole == "" {
			fmt.Println("No muted role set up in", user.GuildID)
			continue
		}

		err = s.GuildMemberRoleRemove(user.GuildID, user.UserID, mutedRole)
		if err != nil {
			if rErr, ok := err.(*discordgo.RESTError); ok {
				if rErr.Message != nil {
					dErr := rErr.Message
					if dErr.Code == 10007 {
						// User not in server anymore
					} else {
						fmt.Println("4 Error removing role", err, dErr.Code)
						continue
					}
				} else {
					fmt.Println("5 Error removing role", err)
					continue
				}
			} else {
				fmt.Println("6 Error removing role", err)
				continue
			}
		}

		unmutedUsers = append(unmutedUsers, user)

		rowsToRemove = append(rowsToRemove, id)
	}

	return
}

func ReapplyMute(s *discordgo.Session, sqlClient *sql.DB, m *discordgo.GuildMemberAdd) error {
	muted, err := IsUserMuted(sqlClient, m.GuildID, m.User.ID)
	if err != nil {
		fmt.Println("Error checking user mute:", err)
		return err
	}

	if muted {
		mutedRole := roles.GetSingle(m.GuildID, "muted")
		if mutedRole == "" {
			return errors.New("no muted role set up")
		}

		// Reapply muted role
		err = s.GuildMemberRoleAdd(m.GuildID, m.User.ID, mutedRole)
		if err != nil {
			return err
		}
	}

	return nil
}
