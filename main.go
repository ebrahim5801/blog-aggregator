package main

import (
	"fmt"
	"os"

	"github.com/ebrahim5801/blog-aggregator/internal/command"
	"github.com/ebrahim5801/blog-aggregator/internal/config"
	"github.com/ebrahim5801/blog-aggregator/internal/state"
)

func main() {
	data, err := config.Read()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	s := &state.State{Config: data}

	if len(os.Args) < 2 {
		fmt.Println("not enough arguments, a command is required")
		os.Exit(1)
	}

	cmds := &command.Commands{}
	cmds.Register("login", handlerLogin)

	cmd := command.Command{
		Name: os.Args[1],
		Args: os.Args[2:],
	}

	if err := cmds.Run(s, cmd); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func handlerLogin(s *state.State, cmd command.Command) error {
	if len(cmd.Args) == 0 {
		return fmt.Errorf("please enter username")
	}

	_, err := s.Config.SetUser(cmd.Args[0])
	if err != nil {
		return err
	}

	fmt.Println("User has been set")
	return nil
}
