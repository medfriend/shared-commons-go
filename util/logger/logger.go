package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

var Logger *zap.Logger

// InitLogger inicializa el logger de zap con configuración para desarrollo y producción
func InitLogger() {
	env := os.Getenv("ENVIRONMENT")

	var config zap.Config

	if env == "production" {
		config = zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	} else {
		config = zap.NewDevelopmentConfig()
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Configuración común
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	var err error
	Logger, err = config.Build()
	if err != nil {
		panic(err)
	}
}

// GetLogger retorna la instancia del logger
func GetLogger() *zap.Logger {
	if Logger == nil {
		InitLogger()
	}
	return Logger
}

// SyncLogger hace flush de los logs pendientes
func SyncLogger() {
	if Logger != nil {
		Logger.Sync()
	}
}
