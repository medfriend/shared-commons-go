package gorm

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/medfriend/shared-commons-go/util/consul"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"strconv"
	"time"
)

func HandleString2int(data string) int {
	converted, err := strconv.Atoi(data)

	if err != nil {
		log.Fatal(err)
	}

	return converted
}

func InitDB(db *gorm.DB, consulClient *api.Client) (*gorm.DB, error) {

	dbString, _ := consul.GetKeyValue(consulClient, "DB")

	var result map[string]string

	err := json.Unmarshal([]byte(dbString), &result)

	if err != nil {
		log.Fatalf("Error converting JSON string to map: %v", err)
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC preferSimpleProtocol=true",
		result["DB_HOST"], result["DB_USER"], result["DB_PASSWORD"], result["DB_NAME"], result["DB_PORT"])

	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("No se pudo conectar a la base de datos: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Error obteniendo la conexión de base de datos: %v", err)
	}

	sqlDB.SetMaxOpenConns(HandleString2int(result["MAX_OPEN_CONN"]))                       // Máximo número de conexiones abiertas
	sqlDB.SetMaxIdleConns(HandleString2int(result["MAX_IDLE_CONN"]))                       // Máximo número de conexiones inactivas
	sqlDB.SetConnMaxLifetime(time.Duration(HandleString2int(result["MAX_LIFETIME_CONN"]))) // Tiempo de vida máximo de una conexión

	fmt.Println("Conexión a la base de datos exitosa")

	return db, nil
}
