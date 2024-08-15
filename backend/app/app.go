package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/kingwrcy/moments/appconfig"
	"github.com/kingwrcy/moments/db"
	_ "github.com/kingwrcy/moments/docs"
	"github.com/kingwrcy/moments/handler"
	"github.com/kingwrcy/moments/logger"
	"github.com/kingwrcy/moments/vo"
	"github.com/labstack/echo/v4"
	"github.com/samber/do/v2"
	echoSwagger "github.com/swaggo/echo-swagger"
	"github.com/tidwall/gjson"
	"gorm.io/gorm"
)

type App struct {
	server           *echo.Echo
	injector         do.Injector
	cfg              *appconfig.AppConfig
	db               *db.DB
	logger           *logger.Logger
	userHandler      *handler.UserHandler
	memoHandler      *handler.MemoHandler
	commentHandler   *handler.CommentHandler
	sysConfigHandler *handler.SysConfigHandler
	fileHandler      *handler.FileHandler
	tagHandler       *handler.TagHandler
}

func (a *App) handleEmptyConfig() {
	currentDir, err := os.Getwd()
	if err != nil {
		a.logger.Error().Msgf("获取当前工作目录异常:%s", err.Error())
		return
	}
	a.logger.Debug().Str("数据库[DB]", a.cfg.DB).
		Int("端口[PORT]", a.cfg.Port).
		Str("JWT密钥[JWT_KEY]", a.cfg.JwtKey).
		Str("上传目录[UPLOAD_DIR]", a.cfg.UploadDir).
		Str("日志级别[LOG_LEVEL]", a.cfg.LogLevel).
		Bool("是否启用Swagger文档[ENABLE_SWAGGER]", a.cfg.EnableSwagger).
		Bool("是否输出SQL[ENABLE_SQL_OUTPUT]", a.cfg.EnableSQLOutput).
		Msgf("基本信息")
	if a.cfg.DB != "" && a.cfg.UploadDir != "" {
		return
	}
	a.logger.Debug().Msgf("没有配置默认所必需的环境变量,使用当前目录[%s]作为项目目录", currentDir)

	if a.cfg.DB == "" {
		a.cfg.DB = filepath.Join(currentDir, "db.sqlite")
		if _, err = os.Stat(a.cfg.DB); err != nil {
			a.logger.Debug().Msgf("当前目录[%s]没有[db.sqlite]数据库文件,自动生成成功", currentDir)
		}
	}
	if a.cfg.UploadDir == "" {
		a.cfg.UploadDir = filepath.Join(currentDir, "upload")
		if _, err = os.Stat(a.cfg.UploadDir); err != nil {
			err = os.MkdirAll(a.cfg.UploadDir, 0755)
			if err != nil {
				a.logger.Fatal().Msgf("创建upload文件夹异常:%s", err.Error())
			} else {
				a.logger.Debug().Msgf("没有配置[上传目录-upload文件夹],在当前目录[%s]生成[upload]文件夹成功", currentDir)
			}
		}
	}
	if a.cfg.JwtKey == "" {
		a.cfg.JwtKey = strings.ReplaceAll(uuid.NewString(), "-", "")
		a.logger.Debug().Msgf("JWT_KEY没有配置,随机生成为%s,每次重启服务需要重新登录,配置后则不会", a.cfg.JwtKey)
	}
}

func (a *App) migrateTo3() {
	var (
		count int64
		admin db.User
		item  vo.FullSysConfigVO
	)
	a.db.Table("SysConfig").Count(&count)
	if count == 0 {
		a.logger.Info().Msg("初始化默认配置...")
		if err := a.db.First(&admin).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			admin.Username = "admin"
			admin.Password = "$2a$12$Ruw0XIDW3IuHmD3WXsRTnOUt/0sfqgKWP3wbsqx5sGcCuebWa6X.i"
			admin.Title = "极简朋友圈"
			admin.Slogan = "修道者，逆天而行，注定要一生孤独。"
			admin.Nickname = "admin"
			admin.EnableS3 = "0"
			admin.Favicon = "/favicon.png"
			admin.CoverUrl = "/cover.webp"
			admin.AvatarUrl = "/avatar.webp"
			if err := a.db.Save(&admin).Error; err != nil {
				a.logger.Info().Msgf("用户不存在,初始化[admin/a123456]用户... 失败:%s", err)
			} else {
				a.logger.Info().Msg("用户不存在,初始化[admin/a123456]用户... 成功!")
			}
		}
		item.AdminUserName = admin.Username
		item.Css = admin.Css
		item.Js = admin.Js
		item.BeiAnNo = admin.BeianNo
		item.Favicon = admin.Favicon
		item.Title = admin.Title
		if admin.EnableS3 == "0" {
			item.EnableS3 = false
		} else {
			item.EnableS3 = true
			item.S3 = vo.S3VO{
				Domain:          admin.Domain,
				Bucket:          admin.Bucket,
				Region:          admin.Region,
				AccessKey:       admin.AccessKey,
				SecretKey:       admin.SecretKey,
				Endpoint:        admin.Endpoint,
				ThumbnailSuffix: admin.ThumbnailSuffix,
			}
		}
		item.EnableGoogleRecaptcha = false
		item.EnableComment = true
		item.MaxCommentLength = 120
		item.MaxCommentLength = 300
		item.CommentOrder = "desc"
		item.TimeFormat = "timeAgo"
		var sysConfig db.SysConfig

		content, err := json.Marshal(&item)
		if err != nil {
			a.logger.Error().Msgf("初始化默认配置执行异常:%s", err)
		}
		sysConfig.Content = string(content)
		if err := a.db.Save(&sysConfig).Error; err == nil {
			a.logger.Info().Msg("初始化默认配置执行成功")
		}

		var memos []db.Memo
		a.db.Find(&memos)
		for _, memo := range memos {
			a.logger.Info().Msgf("开始迁移memo id:%d", memo.Id)
			var extMap = map[string]interface{}{}
			var ext vo.MemoExt
			err := json.Unmarshal([]byte(memo.Ext), &extMap)
			if err != nil {
				a.logger.Warn().Msgf("memo id:%d ext属性不是标准的json格式 => %s,忽略..", memo.Id, memo.Ext)
				continue
			}
			if value, exist := extMap["videoUrl"]; exist && value != "" {
				ext.Video.Type = "online"
				ext.Video.Value = value.(string)
			}
			if value, exist := extMap["localVideoUrl"]; exist && value != "" {
				ext.Video.Type = "online"
				ext.Video.Value = value.(string)
			}
			if value, exist := extMap["youtubeUrl"]; exist && value != "" {
				ext.Video.Type = "youtube"
				ext.Video.Value = value.(string)
			}
			if memo.BilibiliUrl != "" {
				ext.Video.Type = "bilibili"
				ext.Video.Value = fmt.Sprintf("<iframe src=\"%s\" scrolling=\"no\" border=\"0\" frameborder=\"no\" framespacing=\"0\" allowfullscreen=\"true\"></iframe>", memo.BilibiliUrl)
			}
			if value, exist := extMap["doubanBook"]; exist && value != nil {
				val := gjson.Get(memo.Ext, "doubanBook")
				ext.DoubanBook.Title = val.Get("title").Str
				ext.DoubanBook.Desc = val.Get("desc").Str
				ext.DoubanBook.Image = val.Get("image").Str
				ext.DoubanBook.Author = val.Get("author").Str
				ext.DoubanBook.Isbn = val.Get("isbn").Str
				ext.DoubanBook.Url = val.Get("url").Str
				ext.DoubanBook.Rating = val.Get("rating").Str
				ext.DoubanBook.PubDate = val.Get("pubDate").Str
				ext.DoubanBook.Id = val.Get("id").Str
			}
			if value, exist := extMap["doubanMovie"]; exist && value != nil {
				val := gjson.Get(memo.Ext, "doubanMovie")
				ext.DoubanMovie.Title = val.Get("title").Str
				ext.DoubanMovie.Desc = val.Get("desc").Str
				ext.DoubanMovie.Image = val.Get("image").Str
				ext.DoubanMovie.Director = val.Get("director").Str
				ext.DoubanMovie.ReleaseDate = val.Get("releaseDate").Str
				ext.DoubanMovie.Url = val.Get("url").Str
				ext.DoubanMovie.Rating = val.Get("rating").Str
				ext.DoubanMovie.Actors = val.Get("actors").Str
				ext.DoubanMovie.Id = val.Get("id").Str
			}

			extContent, _ := json.Marshal(ext)
			memo.Ext = string(extContent)
			newTags := ""

			memoContent, tags := handler.FindAndReplaceTags(memo.Content)
			if len(tags) > 0 {
				memo.Content = memoContent
				newTags = strings.Join(tags, ",")
				if newTags != "" {
					newTags = newTags + ","
				}
				memo.Tags = &newTags
			}

			if err = a.db.Save(&memo).Error; err != nil {
				a.logger.Info().Msgf("迁移memo id:%d 成功", memo.Id)
			}
		}
	}

	// 修复之前版本的时间格式问题
	a.db.Exec(`UPDATE memo
SET 
    createdAt = datetime(createdAt / 1000, 'unixepoch'),
    updatedAt = datetime(updatedAt / 1000, 'unixepoch')
WHERE 
    ((createdAt NOT LIKE '%-%' AND length(createdAt) = 13) OR 
    (updatedAt NOT LIKE '%-%' AND length(updatedAt) = 13))`)

	a.db.Exec(`UPDATE comment
SET 
    createdAt = datetime(createdAt / 1000, 'unixepoch'),
    updatedAt = datetime(updatedAt / 1000, 'unixepoch')
WHERE 
    ((createdAt NOT LIKE '%-%' AND length(createdAt) = 13) OR 
    (updatedAt NOT LIKE '%-%' AND length(updatedAt) = 13))`)

	a.db.Exec(`UPDATE user
SET 
    createdAt = datetime(createdAt / 1000, 'unixepoch'),
    updatedAt = datetime(updatedAt / 1000, 'unixepoch')
WHERE 
    ((createdAt NOT LIKE '%-%' AND length(createdAt) = 13) OR 
    (updatedAt NOT LIKE '%-%' AND length(updatedAt) = 13))`)
}

func (a *App) setupRouter() {
	api := a.server.Group("/api")

	userGroup := api.Group("/user")
	{
		userGroup.POST("/login", a.userHandler.Login)
		userGroup.POST("/reg", a.userHandler.Reg)
		userGroup.POST("/profile", a.userHandler.Profile)
		userGroup.POST("/profile/:username", a.userHandler.ProfileForUser)
		userGroup.POST("/saveProfile", a.userHandler.SaveProfile)
	}

	memoGroup := api.Group("/memo")
	{
		memoGroup.POST("/list", a.memoHandler.ListMemos)
		memoGroup.POST("/save", a.memoHandler.SaveMemo)
		memoGroup.POST("/remove", a.memoHandler.RemoveMemo)
		memoGroup.POST("/like", a.memoHandler.LikeMemo)
		memoGroup.POST("/get", a.memoHandler.GetMemo)
		memoGroup.POST("/setPinned", a.memoHandler.SetPinned)
		memoGroup.POST("/getFaviconAndTitle", a.memoHandler.GetFaviconAndTitle)
		memoGroup.POST("/getDoubanMovieInfo", a.memoHandler.GetDoubanMovieInfo)
		memoGroup.POST("/getDoubanBookInfo", a.memoHandler.GetDoubanBookInfo)
		memoGroup.POST("/removeImage", a.memoHandler.RemoveImage)
	}

	commentGroup := api.Group("/comment")
	{
		commentGroup.POST("/add", a.commentHandler.AddComment)
		commentGroup.POST("/remove", a.commentHandler.RemoveComment)
	}

	sycConfigGroup := api.Group("/sysConfig")
	{
		sycConfigGroup.POST("/save", a.sysConfigHandler.SaveConfig)
		sycConfigGroup.POST("/get", a.sysConfigHandler.GetConfig)
		sycConfigGroup.POST("/getFull", a.sysConfigHandler.GetFullConfig)
	}

	tagGroup := api.Group("/tag")
	{
		tagGroup.POST("/list", a.tagHandler.List)
	}

	a.server.GET("/upload/:filename", a.fileHandler.Get)
	a.server.POST("/api/file/upload", a.fileHandler.Upload)
	a.server.POST("/api/file/s3PreSigned", a.fileHandler.S3PreSigned)

	if a.cfg.EnableSwagger {
		a.server.GET("/swagger/*", echoSwagger.WrapHandler)
	}
}

func NewApp(i do.Injector) (*App, error) {
	return &App{
		server:           echo.New(),
		injector:         i,
		cfg:              do.MustInvoke[*appconfig.AppConfig](i),
		db:               do.MustInvoke[*db.DB](i),
		logger:           do.MustInvoke[*logger.Logger](i),
		userHandler:      do.MustInvoke[*handler.UserHandler](i),
		memoHandler:      do.MustInvoke[*handler.MemoHandler](i),
		commentHandler:   do.MustInvoke[*handler.CommentHandler](i),
		sysConfigHandler: do.MustInvoke[*handler.SysConfigHandler](i),
		fileHandler:      do.MustInvoke[*handler.FileHandler](i),
		tagHandler:       do.MustInvoke[*handler.TagHandler](i),
	}, nil
}

func init() {
	do.Provide(nil, NewApp)
}
