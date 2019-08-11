package commands

import (
	"fmt"

	"github.com/pajbot/commandmatcher"
	"github.com/pajbot/pajbot2-discord/pkg"
)

var (
	c = commandmatcher.New()
)

func Register(aliases []string, command pkg.Command) {
	if command == nil {
		fmt.Println("Error registering command", aliases)
		return
	}
	c.Register(aliases, command)
}

func Match(text string) (interface{}, []string) {
	return c.Match(text)
}
