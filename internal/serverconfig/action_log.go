package serverconfig

import (
	"database/sql"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/pajbot/pajbot2-discord/internal/values"
)

// GetActionLogIgnoredChannels returns the configured channel IDs whose action-log events should be skipped
func GetActionLogIgnoredChannels(guildID string) []string {
	v := GetValue(guildID, values.ActionLogIgnoredChannels)
	if v == "" {
		return nil
	}

	channelIDs := []string{}

	for rawChannelID := range strings.SplitSeq(v, ",") {
		channelID, err := normalizeActionLogIgnoredChannel(rawChannelID)
		if err != nil {
			fmt.Printf("Skipping invalid ignored action-log channel for guild %s: %s\n", guildID, err)
			continue
		}

		channelIDs = append(channelIDs, channelID)
	}

	return channelIDs
}

// ActionLogChannelIgnored returns true if the given channel ID should be ignored
func ActionLogChannelIgnored(guildID, channelID string) bool {
	ignoredChannels := GetActionLogIgnoredChannels(guildID)
	if len(ignoredChannels) == 0 {
		return false
	}

	return !slices.Contains(ignoredChannels, channelID)
}

// SetActionLogIgnoredChannels validates and stores the action-log ignore list
func SetActionLogIgnoredChannels(sqlClient *sql.DB, guildID string, rawChannelIDs []string) error {
	channelIDs, err := normalizeActionLogIgnoredChannels(rawChannelIDs)
	if err != nil {
		return err
	}

	return SetValue(sqlClient, guildID, values.ActionLogIgnoredChannels, strings.Join(channelIDs, ","))
}

// RemoveActionLogIgnoredChannels clears the configured action-log ignore list for a guild
func RemoveActionLogIgnoredChannels(sqlClient *sql.DB, guildID string) error {
	return RemoveValue(sqlClient, guildID, values.ActionLogIgnoredChannels)
}

func normalizeActionLogIgnoredChannels(rawChannelIDs []string) ([]string, error) {
	channelIDs := make([]string, 0, len(rawChannelIDs))
	seen := make(map[string]struct{}, len(rawChannelIDs))

	for _, rawChannelID := range rawChannelIDs {
		channelID, err := normalizeActionLogIgnoredChannel(rawChannelID)
		if err != nil {
			return nil, err
		}

		if _, ok := seen[channelID]; ok {
			continue
		}

		seen[channelID] = struct{}{}
		channelIDs = append(channelIDs, channelID)
	}

	if len(channelIDs) == 0 {
		return nil, fmt.Errorf("must set at least 1 channel")
	}

	return channelIDs, nil
}

func normalizeActionLogIgnoredChannel(rawChannelID string) (string, error) {
	channelID := strings.TrimSpace(rawChannelID)
	if channelID == "" {
		return "", fmt.Errorf("channel ID must not be empty")
	}

	if _, err := strconv.ParseUint(channelID, 10, 64); err != nil {
		return "", fmt.Errorf("invalid channel ID %q", rawChannelID)
	}

	return channelID, nil
}
