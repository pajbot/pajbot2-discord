package filter

import "context"

type Filter interface {
	Check(ctx context.Context, message Message) (actions []Action, stop bool)
}
