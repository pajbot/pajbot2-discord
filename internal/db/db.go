package db

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/pajbot2-discord/internal/config"
	"github.com/pajlada/stupidmigration"
)

var SQLClient *sql.DB

func init() {
	var err error
	SQLClient, err = sql.Open("postgres", config.DSN)
	if err != nil {
		fmt.Println("Unable to connect to mysql", err)
		os.Exit(1)
	}

	err = SQLClient.Ping()
	if err != nil {
		fmt.Println("Unable to ping mysql", err)
		os.Exit(1)
	}

	err = stupidmigration.Migrate("migrations", SQLClient)
	if err != nil {
		fmt.Println("Unable to run SQL migrations", err)
		os.Exit(1)
	}
}

func PushMessageIntoDatabase(m *discordgo.Message) (err error) {
	const query = `INSERT INTO discord_messages (id, content, author_id) VALUES ($1, $2, $3) ON CONFLICT (id) DO UPDATE SET content=$2`
	authorID := "unknown"
	if m.Author != nil {
		authorID = m.Author.ID
	}
	_, err = SQLClient.Exec(query, m.ID, m.Content, authorID)
	return
}

func PushThreadMessageIntoDatabase(m *discordgo.Message, threadTitle string) (err error) {
	const query = `INSERT INTO discord_messages (id, content, author_id, thread_title) VALUES ($1, $2, $3, $4) ON CONFLICT (id) DO UPDATE SET content=$2, thread_title=$4`
	authorID := "unknown"
	if m.Author != nil {
		authorID = m.Author.ID
	}
	_, err = SQLClient.Exec(query, m.ID, m.Content, authorID, threadTitle)
	return
}

func GetMessageFromDatabase(messageID string) (content string, authorID string, err error) {
	const query = `SELECT content, author_id FROM discord_messages WHERE id=$1`
	row := SQLClient.QueryRow(query, messageID)
	err = row.Scan(&content, &authorID)
	return
}

func GetThreadMessageFromDatabase(messageID string) (content string, authorID string, threadTitle string, err error) {
	const query = `SELECT content, author_id, thread_title FROM discord_messages WHERE id=$1`
	row := SQLClient.QueryRow(query, messageID)
	err = row.Scan(&content, &authorID, &threadTitle)
	return
}
