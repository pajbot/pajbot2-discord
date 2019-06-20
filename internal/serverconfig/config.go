package serverconfig

import "sync"

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

func Set(serverID, key, newChannelID string) {
	fullKey := serverID + ":" + key

	mutex.Lock()
	defer mutex.Unlock()

	data[fullKey] = newChannelID
}
