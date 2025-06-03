package env

import (
	"github.com/joho/godotenv"
	"github.com/medfriend/shared-commons-go/util/logger"
	"go.uber.org/zap"
)

func LoadEnv() {
	err := godotenv.Load(".env")
	if err != nil {
		logger.GetLogger().Fatal("Error loading .env file", zap.Error(err))
	}
}
