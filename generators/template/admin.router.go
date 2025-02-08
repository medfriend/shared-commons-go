package template

func GetAdminRouter(args []string) string {
	return `package router

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RouterInitFunc func(router *gin.RouterGroup, db *gorm.DB)

var routerInits []RouterInitFunc

func RegisterRouter(initFunc RouterInitFunc) {
	routerInits = append(routerInits, initFunc)
}

func InitializeAllRouters(api *gin.RouterGroup, db *gorm.DB) {
	for _, initFunc := range routerInits {
		initFunc(api, db)
	}
}`
}
