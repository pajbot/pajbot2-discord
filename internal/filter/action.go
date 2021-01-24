package filter

import (
	"context"
	"fmt"
)

type Action struct {
	Mute   bool
	Ban    bool
	Delete bool
}

func returnActions(ctx context.Context, actions []Action) []Action {
	if len(actions) == 0 {
		return actions
	}

	if dry, ok := ctx.Value("dry").(bool); ok && dry {
		fmt.Printf("Skipping actions: %#v\n", actions)
		return nil
	}

	return actions
}
