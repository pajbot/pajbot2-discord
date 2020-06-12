package utils

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"regexp"
)

var (
	markdownRegex = regexp.MustCompile(`[_*~|]{1,2}|\\x60{1,3}`)
)

func EscapeMarkdown(s string) string {
	return markdownRegex.ReplaceAllString(s, "\\\\$0")
}

func MentionMember(member *discordgo.Member) string {
	// user is member of guild. We can mention them with <@!ID> to display their nickname,
	// and we can retrieve their guild-specific nickname to display

	var nickOrName = member.Nick
	if nickOrName == "" { // no nickname, use their username
		nickOrName = member.User.Username
	}

	// @pajlada (pajlada#2107 - 646701695224643614)
	return fmt.Sprintf("%s (%s#%s - %s)", member.Mention(), EscapeMarkdown(nickOrName), member.User.Discriminator, member.User.ID)
}

func MentionUser(s *discordgo.Session, guildID string, user *discordgo.User) string {
	var member, err = s.GuildMember(guildID, user.ID)
	if err == nil {
		return MentionMember(member)
	}

	// user is not member of the guild, fall back to more basic information
	return fmt.Sprintf("%s (%s#%s - %s)", user.Mention(), user.Username, user.Discriminator, user.ID)
}
