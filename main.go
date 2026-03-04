package main

import (
	"context"
	"database/sql"
	"fmt"
	"html"
	"os"
	"strconv"
	"strings"
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
	cmds.Register("browse", middlewareLoggedIn(handlerBrowse))
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
	if len(cmd.Args) == 0 {
		return fmt.Errorf("usage: agg <time_between_reqs>")
	}

	timeBetweenRequests, err := time.ParseDuration(cmd.Args[0])
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", cmd.Args[0], err)
	}

	fmt.Printf("Collecting feeds every %s\n", timeBetweenRequests)

	ticker := time.NewTicker(timeBetweenRequests)
	for ; ; <-ticker.C {
		scrapeFeeds(s)
	}
}

var pubDateFormats = []string{
	time.RFC1123Z,
	time.RFC1123,
	time.RFC3339,
	"Mon, 2 Jan 2006 15:04:05 -0700",
	"Mon, 2 Jan 2006 15:04:05 MST",
	"2006-01-02T15:04:05Z",
	"2006-01-02",
}

func parsePubDate(s string) sql.NullTime {
	s = strings.TrimSpace(s)
	for _, layout := range pubDateFormats {
		if t, err := time.Parse(layout, s); err == nil {
			return sql.NullTime{Time: t, Valid: true}
		}
	}
	return sql.NullTime{}
}

func scrapeFeeds(s *state.State) {
	feed, err := s.Db.GetNextFeedToFetch(context.Background())
	if err != nil {
		fmt.Println("error fetching next feed:", err)
		return
	}

	err = s.Db.MarkFeedFetched(context.Background(), feed.ID)
	if err != nil {
		fmt.Println("error marking feed fetched:", err)
		return
	}

	rssFeed, err := rss.FetchFeed(context.Background(), feed.Url)
	if err != nil {
		fmt.Println("error fetching feed:", err)
		return
	}

	fmt.Printf("Fetching feed: %s\n", feed.Name)
	for _, item := range rssFeed.Channel.Item {
		title := html.UnescapeString(item.Title)
		desc := sql.NullString{}
		if item.Description != "" {
			desc = sql.NullString{String: html.UnescapeString(item.Description), Valid: true}
		}

		_, err := s.Db.CreatePost(context.Background(), database.CreatePostParams{
			ID:          uuid.New(),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Title:       title,
			Url:         item.Link,
			Description: desc,
			PublishedAt: parsePubDate(item.PubDate),
			FeedID:      feed.ID,
		})
		if err != nil {
			if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "duplicate key") {
				continue
			}
			fmt.Println("error saving post:", err)
		}
	}
}

func handlerBrowse(s *state.State, cmd command.Command, user database.User) error {
	limit := int32(2)
	if len(cmd.Args) > 0 {
		n, err := strconv.Atoi(cmd.Args[0])
		if err != nil {
			return fmt.Errorf("invalid limit %q", cmd.Args[0])
		}
		limit = int32(n)
	}

	posts, err := s.Db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  limit,
	})
	if err != nil {
		return err
	}

	for _, post := range posts {
		fmt.Printf("Title: %s\n", post.Title)
		fmt.Printf("URL:   %s\n", post.Url)
		if post.PublishedAt.Valid {
			fmt.Printf("Date:  %s\n", post.PublishedAt.Time.Format("2006-01-02"))
		}
		if post.Description.Valid && post.Description.String != "" {
			fmt.Printf("Desc:  %s\n", post.Description.String)
		}
		fmt.Println("---")
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
