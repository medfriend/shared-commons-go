package main
// @title           medfri-pay
// @version         1.0
// @description     micro de 9000.

// @host            localhost:pay
// @BasePath        /medfri-pay

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Ingresa "Bearer {token}" para autenticar.

// @contact.name    Soporte de API
// @contact.url     http://www.soporte-api.com
// @contact.email   soporte@api.com

// @license.name    MIT
// @license.url     https://opensource.org/licenses/MIT

import (
	"encoding/json"
	"fmt"
	"github.com/medfriend/shared-commons-go/util/consul"
	"github.com/medfriend/shared-commons-go/util/env"
	gormUtil "github.com/medfriend/shared-commons-go/util/gorm"
	"github.com/medfriend/shared-commons-go/util/worker"
	"gorm.io/gorm"
	"net/http"
	"runtime"
	"pay-go/httpServer"
)

var db *gorm.DB

func main() {
	env.LoadEnv()

	consulClient := consul.ConnectToConsulKey("", "PAY")
	serviceInfo, _ := consul.GetKeyValue(consulClient, "PAY")

	var result map[string]string
	err := json.Unmarshal([]byte(serviceInfo), &result)

	numCPUs := runtime.NumCPU()

	taskQueue := make(chan *http.Request, 100)

	stop := make(chan struct{})

	worker.CreateWorkers(numCPUs, stop, taskQueue)

	initDB, err := gormUtil.InitDB(
		db,
		consulClient,
		"LOCAL",
		"PAY",
	)

	httpServer.InitHttpServer(taskQueue, initDB, result)

	if err != nil {
		return
	}
}
