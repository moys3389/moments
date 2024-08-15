package logger

import (
	"os"
	"time"

	"github.com/kingwrcy/moments/appconfig"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
)

type Logger struct {
	*zerolog.Logger
}

func NewLogger(i do.Injector) (*Logger, error) {
	cfg := do.MustInvoke[*appconfig.AppConfig](i)

	var level = zerolog.InfoLevel
	err := level.UnmarshalText([]byte(cfg.LogLevel))
	if err != nil {
		level = zerolog.InfoLevel
	} else {
		zerolog.SetGlobalLevel(level)
	}

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.DateTime}).With().Timestamp().Logger()
	return &Logger{&logger}, nil
}

func init() {
	do.Provide(nil, NewLogger)
}
