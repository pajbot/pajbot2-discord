package serverconfig

import (
	"database/sql"
	"fmt"
	"os"
	"sync"
)

var (
	mutex sync.RWMutex
	data  = map[string]string{}
)

func Get(serverID, key string) string {
	fullKey := serverID + ":" + key

	mutex.RLock()
	defer mutex.RUnlock()
	if channelID, ok := data[fullKey]; ok {
		return channelID
	}

	return ""
}

func GetValue(guildID, valueKey string) string {
	key := "value:" + valueKey

	return Get(guildID, key)
}

func set(serverID, key, newChannelID string) {
	fullKey := serverID + ":" + key

	mutex.Lock()
	defer mutex.Unlock()

	data[fullKey] = newChannelID
}

func remove(serverID, key string) {
	fullKey := serverID + ":" + key

	mutex.Lock()
	defer mutex.Unlock()

	delete(data, fullKey)
}

func Load(sqlClient *sql.DB) {
	// Load channel roles from config
	const query = "SELECT server_id, key, value FROM config"
	rows, err := sqlClient.Query(query)
	if err != nil {
		fmt.Println("Error loading channel roles:", err)
		os.Exit(1)
	}

	for rows.Next() {
		var serverID, key, value string
		err := rows.Scan(&serverID, &key, &value)
		if err != nil {
			fmt.Println("Error scanning channel roles:", err)
			os.Exit(1)
		}

		set(serverID, key, value)
	}
}

// Save sets a value in the database, and then updates the internal store
func Save(sqlClient *sql.DB, serverID, key, value string) (err error) {
	if sqlClient == nil {
		return ErrNoSQLClient
	}
	const query = "INSERT INTO config (server_id, key, value) VALUES ($1, $2, $3) ON CONFLICT (server_id, key) DO UPDATE SET value=$3"
	_, err = sqlClient.Exec(query, serverID, key, value)
	if err != nil {
		return
	}

	// Update internal store
	set(serverID, key, value)

	return
}

// Remove a value from the database, also removing it from the internal store
func Remove(sqlClient *sql.DB, serverID, key string) (err error) {
	if sqlClient == nil {
		return ErrNoSQLClient
	}
	const query = "DELETE FROM config WHERE server_id=$1 AND key=$2"
	_, err = sqlClient.Exec(query, serverID, key)
	if err != nil {
		return
	}

	// Update internal store
	remove(serverID, key)

	return
}
