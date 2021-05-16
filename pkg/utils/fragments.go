package utils

import (
	"fmt"
	"regexp"

	"github.com/bwmarrin/discordgo"
)

var (
	markdownRegex = regexp.MustCompile(`([\\\x60_*~|])`)
)

func EscapeMarkdown(s string) string {
	return markdownRegex.ReplaceAllString(s, `\$1`)
}

func MentionMember(member *discordgo.Member) string {
	// user is member of guild. We can mention them with <@!ID> to display their nickname,
	// and we can retrieve their guild-specific nickname to display

	nickOrName := member.Nick
	if nickOrName == "" { // no nickname, use their username
		nickOrName = member.User.Username
	}

	// @pajlada (pajlada#2107 - 646701695224643614)
	return fmt.Sprintf("%s (%s#%s - %s)", member.Mention(), EscapeMarkdown(nickOrName), member.User.Discriminator, member.User.ID)
}

func MentionUserFromParts(s *discordgo.Session, guildID string, userID string, userName string, userDiscriminator string) string {
	member, err := s.GuildMember(guildID, userID)
	if err == nil {
		return MentionMember(member)
	}

	// user is not member of the guild, fall back to more basic information
	return fmt.Sprintf("<@!%s> (%s#%s - %s)", userID, userName, userDiscriminator, userID)
}

func MentionUser(s *discordgo.Session, guildID string, user *discordgo.User) string {
	return MentionUserFromParts(s, guildID, user.ID, user.Username, user.Discriminator)
}
