package app

import (
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kingwrcy/moments/appconfig"
	"github.com/kingwrcy/moments/db"
	"github.com/kingwrcy/moments/handler"
	"github.com/labstack/echo/v4"
	"github.com/samber/do/v2"
)

type AuthMiddleware struct {
	cfg     *appconfig.AppConfig
	db      *db.DB
	ignores []string
}

func (m *AuthMiddleware) Handler(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !strings.HasPrefix(c.Request().URL.Path, "/api") {
			return next(c)
		}
		tokenStr := c.Request().Header.Get("x-api-token")
		cc := handler.CustomContext{Context: c}
		//zlog.Info().Msgf("token :%s", tokenStr)
		if tokenStr != "" {
			token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
				return []byte(m.cfg.JwtKey), nil
			})

			if err != nil || !token.Valid {
				return handler.FailResp(c, handler.TokenInvalid)
			}

			claims := token.Claims.(jwt.MapClaims)
			//zlog.Info().Msgf("user id :%v", claims["userId"])

			var user db.User
			m.db.Select("username", "nickname", "slogan", "id", "avatarUrl", "coverUrl").First(&user, claims["userId"])
			cc.SetUser(&user)
			return next(cc)
		} else {
			path := c.Request().URL.Path
			for _, url := range m.ignores {
				if path == url {
					return next(cc)
				}
			}
			if strings.HasPrefix(path, "/upload") || strings.HasPrefix(path, "/api/user/profile/") {
				return next(cc)
			}
			return handler.FailResp(c, handler.TokenMissing)
		}
	}
}

func NewAuthMiddleware(i do.Injector) (*AuthMiddleware, error) {
	return &AuthMiddleware{
		cfg: do.MustInvoke[*appconfig.AppConfig](i),
		db:  do.MustInvoke[*db.DB](i),
		ignores: []string{
			"/api/user/reg",
			"/api/user/login",
			"/api/memo/list",
			"/api/user/profile",
			"/api/sysConfig/get",
			"/api/memo/like",
			"/api/comment/add",
			"/api/memo/get",
		},
	}, nil
}

func init() {
	do.Provide(nil, NewAuthMiddleware)
}
