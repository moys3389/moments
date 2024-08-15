package appconfig

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
	_ "github.com/joho/godotenv/autoload"
	"github.com/samber/do/v2"
)

type AppConfig struct {
	Version         string `env:"VERSION"`
	DB              string `env:"DB"`
	Port            int    `env:"PORT" env-default:"37892"`
	JwtKey          string `env:"JWT_KEY"`
	UploadDir       string `env:"UPLOAD_DIR"`
	LogLevel        string `env:"LOG_LEVEL" env-default:"INFO"`
	EnableSwagger   bool   `env:"ENABLE_SWAGGER" env-default:"false"`
	EnableSQLOutput bool   `env:"ENABLE_SQL_OUTPUT" env-default:"false"`
}

func NewConfig(i do.Injector) (*AppConfig, error) {
	var cfg AppConfig
	err := cleanenv.ReadEnv(&cfg)
	if err != nil {
		fmt.Printf("读取配置文件异常:%s", err)
		return nil, err
	}

	return &cfg, nil
}

func init() {
	do.Provide(nil, NewConfig)
}
