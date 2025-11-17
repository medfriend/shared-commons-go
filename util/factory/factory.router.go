package factory

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func NewFactoryRouter(router *gin.RouterGroup, db *gorm.DB) {
	factoryController := InitializeFactoryModule(db)

	group := router.Group("factory")

	group.POST("/CreateTable", factoryController.CreateTable)
	group.POST("/Insert/:table", factoryController.Insert)
	group.POST("/Query/:table", factoryController.Query)
    group.POST("/Update/:table", factoryController.Update)
	group.DELETE("/Delete/:table", factoryController.Delete)
	group.DELETE("/DropTable/:table", factoryController.DropTable)
	group.POST("/Relation", factoryController.Relation)
}
