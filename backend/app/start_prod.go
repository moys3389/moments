//go:build prod

package app

import (
	"embed"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4/middleware"
	"github.com/samber/do/v2"
)

//go:embed public/*
var staticFiles embed.FS

func (a *App) Start() error {
	a.handleEmptyConfig()
	a.migrateTo3()
	a.server.Use(do.MustInvoke[*AuthMiddleware](a.injector).Handler)
	a.setupRouter()

	a.server.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		HTML5:      true,
		Root:       "public", // because files are located in `web` directory in `webAssets` fs
		Filesystem: http.FS(staticFiles),
	}))

	a.server.FileFS("/*", "public/index.html", staticFiles)

	a.logger.Info().Msgf("服务端启动成功,监听:%d端口...", a.cfg.Port)
	a.server.HideBanner = true

	err := a.server.Start(fmt.Sprintf(":%d", a.cfg.Port))
	if err != nil {
		a.logger.Fatal().Msgf("服务启动失败,错误原因:%s", err)
	}
	return err
}
