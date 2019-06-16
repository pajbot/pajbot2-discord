package utils

import (
	"regexp"

	"github.com/bwmarrin/discordgo"
)

// MemberInRoles returns true if the given user id is in one of the given roles
func MemberInRoles(s *discordgo.Session, guildID string, userID string, roles []string) (bool, error) {
	member, err := s.State.Member(guildID, userID)
	if err != nil {
		if member, err = s.GuildMember(guildID, userID); err != nil {
			return false, err
		}
	}

	// Iterate through the role IDs stored in member.Roles
	// to check permissions
	for _, roleID := range member.Roles {
		role, err := s.State.Role(guildID, roleID)
		if err != nil {
			return false, err
		}
		for _, tRole := range roles {
			if role.ID == tRole {
				return true, nil
			}
		}
	}

	return false, nil
}

// GetChannelTypeName returns a readable name for a discordgo.ChannelType
func GetChannelTypeName(channelType discordgo.ChannelType) string {
	switch channelType {
	case discordgo.ChannelTypeGuildCategory:
		return "Category"
	case discordgo.ChannelTypeGuildText:
		return "Text"
	case discordgo.ChannelTypeGuildVoice:
		return "Voice"
	case discordgo.ChannelTypeDM:
		return "DM"
	case discordgo.ChannelTypeGroupDM:
		return "Group DM"
	default:
		return "unknown"
	}
}

var patternUserIDReplacer = regexp.MustCompile(`^<@!?([0-9]+)>$`)
var patternUserID = regexp.MustCompile(`^[0-9]+$`)

func CleanUserID(input string) string {
	output := patternUserIDReplacer.ReplaceAllString(input, "$1")

	if !patternUserID.MatchString(output) {
		return ""
	}

	return output
}
