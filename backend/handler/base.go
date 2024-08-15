package handler

import (
	"github.com/kingwrcy/moments/appconfig"
	"github.com/kingwrcy/moments/db"
	"github.com/kingwrcy/moments/logger"
	"github.com/samber/do/v2"
)

type BaseHandler struct {
	injector do.Injector
	cfg      *appconfig.AppConfig
	db       *db.DB
	logger   *logger.Logger
}

func NewBaseHandler(injector do.Injector) (*BaseHandler, error) {
	return &BaseHandler{
		injector: injector,
		cfg:      do.MustInvoke[*appconfig.AppConfig](injector),
		db:       do.MustInvoke[*db.DB](injector),
		logger:   do.MustInvoke[*logger.Logger](injector),
	}, nil
}

func init() {
	do.Provide(nil, NewBaseHandler)
}
