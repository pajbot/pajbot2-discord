package filter

import (
	"context"
	"fmt"
)

type Runner struct {
	messages chan Message

	filters []Filter

	ctx context.Context

	Dry bool

	onMuteCB   func(message Message)
	onBanCB    func(message Message)
	onDeleteCB func(message Message)
}

func NewRunner() *Runner {
	return &Runner{
		messages: make(chan Message, 100),
	}
}

func (r *Runner) AddFilter(f Filter) {
	r.filters = append(r.filters, f)
}

func (r *Runner) Run(ctx context.Context) {
	r.ctx = context.WithValue(ctx, "dry", r.Dry)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("doner xd")
			return

		case message := <-r.messages:
			r.scanMessage(message)
		}
	}
}

func (r *Runner) OnMute(cb func(message Message)) {
	r.onMuteCB = cb
}

func (r *Runner) onMute(message Message) {
	if r.onMuteCB != nil {
		r.onMuteCB(message)
	}
}

func (r *Runner) OnBan(cb func(message Message)) {
	r.onBanCB = cb
}

func (r *Runner) onBan(message Message) {
	if r.onBanCB != nil {
		r.onBanCB(message)
	}
}

func (r *Runner) OnDelete(cb func(message Message)) {
	r.onDeleteCB = cb
}

func (r *Runner) onDelete(message Message) {
	if r.onDeleteCB != nil {
		r.onDeleteCB(message)
	}
}

func (r *Runner) scanMessage(message Message) {
	for _, f := range r.filters {
		actions, stop := f.Check(r.ctx, message)

		for _, action := range actions {
			// Perform actions!
			if action.Mute {
				r.onMute(message)
			}

			if action.Ban {
				r.onBan(message)
			}

			if action.Delete {
				r.onDelete(message)
			}
		}

		if stop {
			return
		}
	}
}

func (r *Runner) ScanMessage(message Message) {
	r.messages <- message
}
