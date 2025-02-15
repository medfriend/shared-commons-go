package template

import (
	"fmt"
	"github.com/medfriend/shared-commons-go/generators/util"
)

func GetModule(args []string) string {
	capitalized := util.CapitalizeFirst(args[0])

	return fmt.Sprintf(`// filename: %s.module.go
// go:build wireinject
//go:build wireinject
// +build wireinject

package module

import (
	"github.com/google/wire"
	"gorm.io/gorm"
	"%s-go/controller"
	"%s-go/repository"
	"%s-go/service"
)

var %sSet = wire.NewSet(
	repository.New%sRepository,
	service.New%sService,
	controller.New%sController)

func Initialize%sModule(db *gorm.DB) *controller.%sController {
	wire.Build(%sSet)
	return nil
}`, args[0], args[1], args[1], args[1], capitalized, capitalized, capitalized,
		capitalized, capitalized, capitalized, capitalized)
}
