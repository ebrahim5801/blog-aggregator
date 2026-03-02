package main

import (
	"context"
	"database/sql"
	"fmt"
	"html"
	"os"
	"time"

	"github.com/ebrahim5801/blog-aggregator/internal/command"
	"github.com/ebrahim5801/blog-aggregator/internal/config"
	"github.com/ebrahim5801/blog-aggregator/internal/database"
	"github.com/ebrahim5801/blog-aggregator/internal/rss"
	"github.com/ebrahim5801/blog-aggregator/internal/state"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func main() {
	data, err := config.Read()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	s := &state.State{Config: data}

	db, err := sql.Open("postgres", s.Config.DBURL)
	if err != nil {
		fmt.Println(err)
	}
	dbQueries := database.New(db)

	s.Db = dbQueries

	if len(os.Args) < 2 {
		fmt.Println("not enough arguments, a command is required")
		os.Exit(1)
	}

	cmds := &command.Commands{}

	cmds.Register("login", handlerLogin)
	cmds.Register("register", handlerRegister)
	cmds.Register("reset", handlerReset)
	cmds.Register("users", handlerUsers)
	cmds.Register("agg", handlerRss)
	cmds.Register("feeds", handlerFeeds)
	cmds.Register("addfeed", middlewareLoggedIn(handlerAddFeed))
	cmds.Register("follow", middlewareLoggedIn(handlerFollow))
	cmds.Register("unfollow", middlewareLoggedIn(handlerUnfollow))
	cmds.Register("following", middlewareLoggedIn(handlerFollowing))

	cmd := command.Command{
		Name: os.Args[1],
		Args: os.Args[2:],
	}

	if err := cmds.Run(s, cmd); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func middlewareLoggedIn(
	handler func(s *state.State, cmd command.Command, user database.User) error,
) func(*state.State, command.Command) error {
	return func(s *state.State, cmd command.Command) error {
		if s.Config.CurrentUserName == "" {
			return fmt.Errorf("you must be logged in to use this command")
		}

		user, err := s.Db.GetUser(context.Background(), s.Config.CurrentUserName)
		if err != nil {
			return err
		}

		return handler(s, cmd, user)
	}
}

func handlerLogin(s *state.State, cmd command.Command) error {
	if len(cmd.Args) == 0 {
		return fmt.Errorf("please enter username")
	}
	user, err := s.Db.GetUser(context.Background(), cmd.Args[0])
	if err != nil {
		os.Exit(1)
		return err
	}
	_, err = s.Config.SetUser(cmd.Args[0])
	if err != nil {
		return err
	}
	fmt.Println(user)
	return nil
}

func handlerRegister(s *state.State, cmd command.Command) error {
	if len(cmd.Args) == 0 {
		return fmt.Errorf("please enter username")
	}

	_, err := s.Db.GetUser(context.Background(), cmd.Args[0])
	if err != nil {
		user := database.CreateUserParams{
			ID:        uuid.New(),
			Name:      cmd.Args[0],
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		newUser, err := s.Db.CreateUser(context.Background(), user)
		if err != nil {
			return err
		}
		_, err = s.Config.SetUser(cmd.Args[0])
		if err != nil {
			return err
		}

		fmt.Println(newUser)
	} else {
		fmt.Println("user alread exists")
		os.Exit(1)
	}

	return nil
}

func handlerReset(s *state.State, cmd command.Command) error {
	err := s.Db.DeleteUsers(context.Background())
	if err != nil {
		return err
	}
	fmt.Println("Reset complete")

	return nil
}

func handlerUsers(s *state.State, cmd command.Command) error {
	users, err := s.Db.GetUsers(context.Background())
	if err != nil {
		return err
	}
	for _, user := range users {
		fmt.Printf("* %s", user.Name)
		if user.Name == s.Config.CurrentUserName {
			fmt.Print(" (current)")
		}
		fmt.Print("\n")
	}

	return nil
}

func handlerRss(s *state.State, cmd command.Command) error {
	feed, err := rss.FetchFeed(context.Background(), "https://www.wagslane.dev/index.xml")
	if err != nil {
		return err
	}

	for _, item := range feed.Channel.Item {
		fmt.Printf("- '%s'\n", html.UnescapeString(item.Title))
		fmt.Printf("- '%s'\n", html.UnescapeString(item.Description))
	}

	return nil
}

func handlerAddFeed(s *state.State, cmd command.Command, user database.User) error {
	if len(cmd.Args) < 2 {
		return fmt.Errorf("please enter name and url")
	}

	data := database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      cmd.Args[0],
		Url:       cmd.Args[1],
		UserID:    user.ID,
	}

	feed, err := s.Db.CreateFeed(context.Background(), data)
	if err != nil {
		return err
	}

	followData := database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	}

	_, err = s.Db.CreateFeedFollow(context.Background(), followData)
	if err != nil {
		return err
	}

	fmt.Println(feed)
	return nil
}

func handlerFeeds(s *state.State, cmd command.Command) error {
	feeds, err := s.Db.GetFeeds(context.Background())
	if err != nil {
		os.Exit(1)
		return err
	}
	for _, feed := range feeds {
		fmt.Println(feed.Name)
		fmt.Println(feed.UserName)
	}
	return nil
}

func handlerFollow(s *state.State, cmd command.Command, user database.User) error {
	if len(cmd.Args) == 0 {
		return fmt.Errorf("please enter url")
	}

	feed, err := s.Db.GetFeed(context.Background(), cmd.Args[0])
	if err != nil {
		return err
	}

	data := database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	}

	_, err = s.Db.CreateFeedFollow(context.Background(), data)
	if err != nil {
		return err
	}

	fmt.Println(feed.Name)
	return nil
}

func handlerUnfollow(s *state.State, cmd command.Command, user database.User) error {
	if len(cmd.Args) == 0 {
		return fmt.Errorf("please enter url")
	}

	feed, err := s.Db.GetFeed(context.Background(), cmd.Args[0])
	if err != nil {
		return err
	}

	data := database.DeleteFeedFollowParams{
		UserID: user.ID,
		FeedID: feed.ID,
	}

	err = s.Db.DeleteFeedFollow(context.Background(), data)
	if err != nil {
		return err
	}

	return nil
}

func handlerFollowing(s *state.State, cmd command.Command, user database.User) error {
	feeds, err := s.Db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return err
	}

	for _, feed := range feeds {
		fmt.Println(feed.FeedName)
	}

	return nil
}
