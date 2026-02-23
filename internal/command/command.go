package command

import (
	"fmt"

	"github.com/ebrahim5801/blog-aggregator/internal/state"
)

type Command struct {
	Name string
	Args []string
}

type Commands struct {
	Handlers map[string]func(*state.State, Command) error
}

func (c *Commands) Run(s *state.State, cmd Command) error {
	handler, ok := c.Handlers[cmd.Name]
	if !ok {
		return fmt.Errorf("unknown command: %s", cmd.Name)
	}
	return handler(s, cmd)
}

func (c *Commands) Register(name string, f func(*state.State, Command) error) {
	if c.Handlers == nil {
		c.Handlers = make(map[string]func(*state.State, Command) error)
	}
	c.Handlers[name] = f
}
