package router

	import (
    "github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"pay-go/module"
	)

	func NewCardRouter(router *gin.RouterGroup, db *gorm.DB) {

		CardController := module.InitializeCardModule(db)

		routerGroup := router.Group("Card")

		routerGroup.POST("/", CardController.CreateCard)
		routerGroup.GET("/:id", CardController.GetCardById)
		routerGroup.PUT("/:id", CardController.UpdateCard)
		routerGroup.DELETE("/:id", CardController.DeleteCard)
		routerGroup.GET("/all", CardController.GetAllCards)
	}

	func init() {
		RegisterRouter(NewCardRouter)
	}