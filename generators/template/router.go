package template

import (
	"fmt"
	"github.com/medfriend/shared-commons-go/generators/util"
)

func GetRoute(args []string) string {
	capitalized := util.CapitalizeFirst(args[0])

	return fmt.Sprintf(`package router

	import (
    "github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"%s-go/module"
	)

	func New%sRouter(router *gin.RouterGroup, db *gorm.DB) {

		%sController := module.Initialize%sModule(db)

		routerGroup := router.Group("%s")

		routerGroup.POST("/", %sController.Create%s)
		routerGroup.GET("/:id", %sController.Get%sById)
		routerGroup.PUT("/:id", %sController.Update%s)
		routerGroup.DELETE("/:id", %sController.Delete%s)
		routerGroup.GET("/all", %sController.GetAll%ss)
	}

	func init() {
		RegisterRouter(New%sRouter)
	}`, args[1], capitalized, capitalized, capitalized, capitalized, capitalized,
		capitalized, capitalized, capitalized, capitalized, capitalized, capitalized,
		capitalized, capitalized, capitalized, capitalized)
}
