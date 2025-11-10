package factory

import (
	"gorm.io/gorm"
)

// Inicializador sin wire para evitar regeneración de wire_gen.
func InitializeFactoryModule(db *gorm.DB) *FactoryController {
	repo := NewFactoryRepository(db)
	svc := NewFactoryService(repo)
	return NewFactoryController(svc)
}
