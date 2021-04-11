package middlewares

import (
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/microsvc/filters"
)

// Init initialize and load all middlewares to factory
func Init() error {
	logger.Info.Printf("initializing filters")
	filters.SetFilter(NewBlackUserFilter())
	filters.SetFilter(NewBlackAppFilter())
	return nil
}
