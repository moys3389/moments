package db

import (
	"context"
	"errors"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/kingwrcy/moments/appconfig"
	"github.com/kingwrcy/moments/logger"
	"github.com/samber/do/v2"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

type myLog struct {
	SlowThreshold         time.Duration
	SourceField           string
	SkipErrRecordNotFound bool
	logger                *logger.Logger
	cfg                   *appconfig.AppConfig
}

func newMyLog(injector do.Injector) *myLog {
	return &myLog{
		logger:        do.MustInvoke[*logger.Logger](injector),
		SlowThreshold: time.Second,
		SourceField:   "source",
		cfg:           do.MustInvoke[*appconfig.AppConfig](injector),
	}
}

func (m myLog) LogMode(level glogger.LogLevel) glogger.Interface {
	return m
}

func (m myLog) Info(ctx context.Context, s string, i ...interface{}) {
	m.logger.Info().Msgf(s, i)
}

func (m myLog) Warn(ctx context.Context, s string, i ...interface{}) {
	m.logger.Warn().Msgf(s, i)
}

func (m myLog) Error(ctx context.Context, s string, i ...interface{}) {
	m.logger.Error().Msgf(s, i)
}

func (m myLog) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if !m.cfg.EnableSQLOutput {
		return
	}
	elapsed := time.Since(begin)
	sql, _ := fc()
	if err != nil && !(errors.Is(err, gorm.ErrRecordNotFound) && m.SkipErrRecordNotFound) {
		m.logger.Error().Msgf("[GORM] query error,source = %s ,sql = %s,error= %s", utils.FileWithLineNum(), sql, err)
		return
	}

	if m.SlowThreshold != 0 && elapsed > m.SlowThreshold {
		m.logger.Warn().Msgf("[GORM] source = %s slow query sql = %s,耗时 = %s", utils.FileWithLineNum(), sql, elapsed)
		return
	}

	m.logger.Debug().Msgf("[GORM] query source = %s sql = %s,耗时 = %s", utils.FileWithLineNum(), sql, elapsed)
}

type DB struct {
	*gorm.DB
}

func NewDB(injector do.Injector) (*DB, error) {
	cfg := do.MustInvoke[*appconfig.AppConfig](injector)
	logger := do.MustInvoke[*logger.Logger](injector)
	db, err := gorm.Open(sqlite.Open(cfg.DB), &gorm.Config{
		Logger: newMyLog(injector),
	})
	if err != nil {
		logger.Fatal().Msgf("无法连接到数据库: %v,路径:%s", err, cfg.DB)
	} else {
		logger.Debug().Msgf("连接数据库路径:%s,成功", cfg.DB)
	}

	// 迁移 schema
	err = db.AutoMigrate(&User{}, &Comment{}, &Memo{}, &SysConfig{})
	if err != nil {
		return nil, err
	}
	return &DB{db}, nil
}

func init() {
	do.Provide(nil, NewDB)
}
