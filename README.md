# Gator

A CLI blog aggregator that fetches RSS feeds and lets you browse posts from the terminal.

## Prerequisites

- [Go](https://go.dev/doc/install) 1.23+
- [PostgreSQL](https://www.postgresql.org/download/)

## Installation

```bash
go install github.com/ebrahim5801/blog-aggregator@latest
```

## Configuration

Create `~/.gatorconfig.json` with your Postgres connection string:

```json
{
  "db_url": "postgres://username:password@localhost:5432/gator?sslmode=disable"
}
```

## Running the migrations

Apply the database schema using [goose](https://github.com/pressly/goose) or any Postgres migration tool against the files in `sql/schema/`.

## Commands

### User management

```bash
# Register a new user and log in
gator register <username>

# Log in as an existing user
gator login <username>

# List all users
gator users
```

### Feed management

```bash
# Add a feed and automatically follow it (must be logged in)
gator addfeed <name> <url>

# List all feeds
gator feeds

# Follow an existing feed by URL
gator follow <url>

# Unfollow a feed by URL
gator unfollow <url>

# List feeds you are following
gator following
```

### Aggregation

```bash
# Start fetching feeds on an interval (e.g. every 30 seconds)
gator agg 30s
```

### Browsing posts

```bash
# Browse the latest 2 posts from feeds you follow
gator browse

# Browse a specific number of posts
gator browse 10
```
