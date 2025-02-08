package template

import (
	"fmt"
	"strings"
)

func GetMain(args []string) string {

	upperCaseName := strings.ToUpper(args[0])
	return fmt.Sprintf(`package main
// @title           medfri-%s
// @version         1.0
// @description     micro de %s.

// @host            localhost:%s
// @BasePath        /medfri-%s

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
	"%s-go/httpServer"
)

var db *gorm.DB

func main() {
	env.LoadEnv()

	consulClient := consul.ConnectToConsulKey("", "%s")
	serviceInfo, _ := consul.GetKeyValue(consulClient, "%s")

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
		"%s",
	)

	httpServer.InitHttpServer(taskQueue, initDB, result)

	if err != nil {
		return
	}
}
`, args[0], args[1], args[0], args[0], args[0], upperCaseName, upperCaseName, upperCaseName)
}
