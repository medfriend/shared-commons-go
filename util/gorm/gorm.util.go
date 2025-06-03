package gorm

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/medfriend/shared-commons-go/util/consul"
	"github.com/medfriend/shared-commons-go/util/logger"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"strconv"
	"time"
)

func HandleString2int(data string) int {
	converted, err := strconv.Atoi(data)

	if err != nil {
		logger.GetLogger().Fatal("Error al convertir string a int", zap.Error(err))
	}

	return converted
}

func InitDB(db *gorm.DB, consulClient *api.Client, connectionType string, service string) (*gorm.DB, error) {

	dbKey := "DB"

	if connectionType == "LOCAL" {
		dbKey = "DB_LOCAL"
	}

	dbName := fmt.Sprintf("DB_NAME_%s", service)

	dbString, _ := consul.GetKeyValue(consulClient, dbKey)

	var result map[string]string

	err := json.Unmarshal([]byte(dbString), &result)

	if err != nil {
		logger.GetLogger().Fatal("Error converting JSON string to map", zap.Error(err))
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		result["DB_HOST"], result["DB_USER"], result["DB_PASSWORD"], result[dbName], result["DB_PORT"])

	logger.GetLogger().Debug("Conectando a la base de datos", zap.String("dsn", dsn))

	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.GetLogger().Fatal("No se pudo conectar a la base de datos", zap.Error(err))
	}

	sqlDB, err := db.DB()
	if err != nil {
		logger.GetLogger().Fatal("Error obteniendo la conexión de base de datos", zap.Error(err))
	}

	sqlDB.SetMaxOpenConns(HandleString2int(result["MAX_OPEN_CONN"]))                       // Máximo número de conexiones abiertas
	sqlDB.SetMaxIdleConns(HandleString2int(result["MAX_IDLE_CONN"]))                       // Máximo número de conexiones inactivas
	sqlDB.SetConnMaxLifetime(time.Duration(HandleString2int(result["MAX_LIFETIME_CONN"]))) // Tiempo de vida máximo de una conexión

	logger.GetLogger().Info("Conexión a la base de datos exitosa")

	return db, nil
}
