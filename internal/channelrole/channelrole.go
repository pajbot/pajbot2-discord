package channelrole

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pajbot/pajbot2-discord/internal/channels"
	"github.com/pajbot/pajbot2-discord/pkg/commands"
	"github.com/pajbot/pajbot2-discord/pkg/utils"
)

const ChannelRolePingCooldownMinutes = 10 * time.Minute

func CanChannelRolePing(s *discordgo.Session, guildID, userID, channelID, roleID string) (canPing bool, errorMessage string) {
	// Does the user have the role that they are trying to ping?
	userInRole, err := utils.MemberInRoles(s, guildID, userID, roleID)
	if err != nil {
		return false, err.Error()
	}
	if !userInRole {
		return false, "You need to have the role that you are mentioning to ping it."
	}

	// Did the user invoke any at ping command inside this guild in the last 10 minutes?
	inCooldown, errorMessage := IsUserInCooldown(userID, guildID)
	if inCooldown {
		return false, errorMessage
	}

	const query = "SELECT last_invoked, last_invoker_id, channel_id FROM channel_topic_roles WHERE role_id=$1 AND guild_id=$2"

	row := commands.SQLClient.QueryRow(query, roleID, guildID)

	var lastInvoked time.Time
	var lastInvokerID string
	var dbChannelID string

	err = row.Scan(&lastInvoked, &lastInvokerID, &dbChannelID)
	if err != nil {
		return false, err.Error()
	}

	// Was this role pinged within the last 10 minutes?
	if time.Since(lastInvoked) < ChannelRolePingCooldownMinutes {
		return false, "You cannot ping the same role within the last 10 minutes."
	}

	// Was this role pinged in a different channel?
	if dbChannelID != channelID {
		return false, "You cannot ping this role in this channel."
	}

	return true, ""
}

func IsUserInCooldown(userID string, guildID string) (inCooldown bool, errorMessage string) {
	const query = "SELECT last_invoked FROM channel_topic_roles WHERE last_invoker_id=$1 AND guild_id=$2"

	var lastInvoked time.Time

	rows, err := commands.SQLClient.Query(query, userID)
	if err != nil {
		return false, err.Error()
	}

	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&lastInvoked)
		if err != nil {
			return false, err.Error()
		}

		if time.Since(lastInvoked) < ChannelRolePingCooldownMinutes {
			return true, "You are in cooldown. Please wait before pinging again."
		}
	}

	return false, ""
}

func UpdateLastInvoked(s *discordgo.Session, user *discordgo.User, guildID string, roleID string, roleName string) error {
	const updateQuery = "UPDATE channel_topic_roles SET last_invoked=$1, last_invoker_id=$2 WHERE role_id=$3 AND guild_id=$4"

	_, err := commands.SQLClient.Exec(updateQuery, time.Now(), user.ID, roleID, guildID)
	if err != nil {
		return err
	}

	targetChannel, err := getActionLogChannel(guildID)
	if err != nil {
		return err
	}

	message := fmt.Sprintf("Channel role %s (%s) invoked by %s", roleName, roleID, utils.MentionUser(s, guildID, user))

	s.ChannelMessageSend(targetChannel, message)

	return nil
}

func Create(s *discordgo.Session, moderator *discordgo.User, guildID string, channelID string, roleID string, roleName string) error {
	const createQuery = "INSERT INTO channel_topic_roles (role_id, guild_id, channel_id, created_by, last_invoked) VALUES ($1, $2, $3, $4, $5)"
	_, err := commands.SQLClient.Exec(createQuery, roleID, guildID, channelID, moderator.ID, time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		return err
	}

	targetChannel, err := getActionLogChannel(guildID)
	if err != nil {
		return err
	}

	message := fmt.Sprintf("Channel role %s (%s) created by %s", roleName, roleID, utils.MentionUser(s, guildID, moderator))
	s.ChannelMessageSendComplex(targetChannel, &discordgo.MessageSend{
		Content: message,
		AllowedMentions: &discordgo.MessageAllowedMentions{
			Users: []string{},
		},
	})

	return nil
}

func Delete(s *discordgo.Session, moderator *discordgo.User, guildID string, roleID string, roleName string) error {
	const deleteQuery = "DELETE FROM channel_topic_roles WHERE role_id=$1 AND guild_id=$2"

	_, err := commands.SQLClient.Exec(deleteQuery, roleID, guildID)
	if err != nil {
		return err
	}

	targetChannel, err := getActionLogChannel(guildID)
	if err != nil {
		return err
	}

	message := fmt.Sprintf("Channel role %s (%s) deleted by %s", roleName, roleID, utils.MentionUser(s, guildID, moderator))
	s.ChannelMessageSendComplex(targetChannel, &discordgo.MessageSend{
		Content: message,
		AllowedMentions: &discordgo.MessageAllowedMentions{
			Users: []string{},
		},
	})

	return nil
}

func getActionLogChannel(guildID string) (string, error) {
	targetChannel := channels.Get(guildID, "action-log")
	if targetChannel == "" {
		fmt.Println("No channel set up for action log")
		return "", fmt.Errorf("no channel set up for action log")
	}

	return targetChannel, nil
}
