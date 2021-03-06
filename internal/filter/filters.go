package filter

import (
	"context"

	"github.com/bwmarrin/discordgo"
)

type Mute struct {
	Checker func(string) bool
}

func (f *Mute) Check(ctx context.Context, message Message) (actions []Action, stop bool) {
	if f.Checker(message.DiscordMessage.Content) {
		return returnActions(ctx, []Action{
			{
				Mute: true,
			},
		}), true
	}

	return nil, false
}

type Delete struct {
	Checker func(string) bool
}

func (f *Delete) Check(ctx context.Context, message Message) (actions []Action, stop bool) {
	if f.Checker(message.DiscordMessage.Content) {
		return returnActions(ctx, []Action{
			{
				Delete: true,
			},
		}), true
	}

	return nil, false
}

type DeleteAdvanced struct {
	Checker func(*discordgo.Message) bool
}

func (f *DeleteAdvanced) Check(ctx context.Context, message Message) (actions []Action, stop bool) {
	if f.Checker(message.DiscordMessage) {
		return returnActions(ctx, []Action{
			{
				Delete: true,
			},
		}), true
	}

	return nil, false
}
