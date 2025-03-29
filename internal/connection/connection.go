package connection

import (
	"database/sql"
	"fmt"
	"slices"
)

type DiscordConnection struct {
	DiscordUserID   string
	DiscordUserName string
	TwitchUserID    string
	TwitchUserLogin string
}

func Get(sqlClient *sql.DB, guildID, userID string) (*DiscordConnection, error) {
	const query = "SELECT discord_user_name, twitch_user_id, twitch_user_login FROM discord_connection WHERE discord_guild_id=$1 AND discord_user_id=$2"
	discordConnection := DiscordConnection{
		DiscordUserID:   userID,
		DiscordUserName: "",
		TwitchUserID:    "",
		TwitchUserLogin: "",
	}
	fmt.Println("guild id:", guildID, "userid:", userID)
	row := sqlClient.QueryRow(query, guildID, userID)
	err := row.Scan(
		&discordConnection.DiscordUserName,
		&discordConnection.TwitchUserID,
		&discordConnection.TwitchUserLogin,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &discordConnection, nil
}

func GetByTwitchID(sqlClient *sql.DB, twitchUserID, discordGuildID string, discordIDsToIgnore ...string) ([]*DiscordConnection, error) {
	const query = "SELECT discord_user_id, discord_user_name, twitch_user_login FROM discord_connection WHERE twitch_user_id=$1 AND discord_guild_id=$2"
	row := sqlClient.QueryRow(query, twitchUserID, discordGuildID)
	var connections []*DiscordConnection
	for {
		discordConnection := DiscordConnection{
			TwitchUserID: twitchUserID,
		}
		err := row.Scan(
			&discordConnection.DiscordUserID,
			&discordConnection.DiscordUserName,
			&discordConnection.TwitchUserLogin,
		)
		if err != nil {
			if err == sql.ErrNoRows {
				break
			}
			return nil, err
		}

		doIgnore := slices.Contains(discordIDsToIgnore, discordConnection.DiscordUserID)

		if !doIgnore {
			connections = append(connections, &discordConnection)
		}
	}

	return connections, nil
}

func Upsert(sqlClient *sql.DB, discordGuildID, discordUserID, discordUserName, twitchUserID, twitchUserLogin string) error {
	const query = "INSERT INTO discord_connection (discord_guild_id, discord_user_id, discord_user_name, twitch_user_id, twitch_user_login) VALUES($1, $2, $3, $4, $5) ON CONFLICT (discord_guild_id, discord_user_id) DO UPDATE SET discord_user_name=$2, twitch_user_id=$4, twitch_user_login=$5"
	res, err := sqlClient.Exec(query, discordGuildID, discordUserID, discordUserName, twitchUserID, twitchUserLogin)
	fmt.Println("upsert res:", res)
	return err
}
