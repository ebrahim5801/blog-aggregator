package state

import (
	"github.com/ebrahim5801/blog-aggregator/internal/config"
	"github.com/ebrahim5801/blog-aggregator/internal/database"
)

type State struct {
	Db     *database.Queries
	Config *config.Config
}
