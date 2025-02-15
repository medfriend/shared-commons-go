// filename: card.module.go
// go:build wireinject
//go:build wireinject
// +build wireinject

package module

import (
	"github.com/google/wire"
	"gorm.io/gorm"
	"pay-go/controller"
	"pay-go/repository"
	"pay-go/service"
)

var CardSet = wire.NewSet(
	repository.NewCardRepository,
	service.NewCardService,
	controller.NewCardController)

func InitializeCardModule(db *gorm.DB) *controller.CardController {
	wire.Build(CardSet)
	return nil
}