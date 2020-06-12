package utils

import (
	"errors"
	"regexp"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/pajbot2-discord/internal/roles"
)

// MemberAdmin returns true if the given user id is an admin
func MemberAdmin(s *discordgo.Session, guildID, userID string) (bool, error) {
	member, err := s.State.Member(guildID, userID)
	if err != nil {
		if member, err = s.GuildMember(guildID, userID); err != nil {
			return false, err
		}
	}

	guild, err := s.Guild(guildID)
	if err != nil {
		return false, err
	}
	if guild.OwnerID == userID {
		return true, nil
	}

	// Iterate through the role IDs stored in member.Roles
	// to check permissions
	for _, roleID := range member.Roles {
		role, err := s.State.Role(guildID, roleID)
		if err != nil {
			return false, err
		}
		if role.Permissions&discordgo.PermissionAdministrator != 0 {
			return true, nil
		}
	}

	return MemberInRoles(s, guildID, userID, "admin")
}

// MemberInRoles returns true if the given user id is in one of the given roles
func MemberInRoles(s *discordgo.Session, guildID, userID, role string) (bool, error) {
	roles := roles.Get(guildID, role)
	if len(roles) == 0 {
		return false, errors.New("No roles set up")
	}

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

// CreationTime returns the creation time of a Snowflake ID relative to the creation of Discord.
// Taken from https://github.com/Moonlington/FloSelfbot/blob/master/commands/commandutils.go#L117
func CreationTime(ID string) (t time.Time, err error) {
	i, err := strconv.ParseInt(ID, 10, 64)
	if err != nil {
		return

	}
	timestamp := (i >> 22) + 1420070400000
	t = time.Unix(timestamp/1000, 0)
	return
}

var inviteRegex = regexp.MustCompile(`(?i)discord(?:(?:app)?\.com\/invite|\.gg(?:\/invite)?)\/([\w-]{2,255})`)

func ResolveInviteCode(content string) (inviteCode string, ok bool) {
	matches := inviteRegex.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return matches[1], true
	}

	return "", false
}
