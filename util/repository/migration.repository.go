package repository

import (
	"errors"
	"github.com/medfriend/shared-commons-go/util/entity"
	"gorm.io/gorm"
)

type MigrationRepository interface {
	Save(Migration *entity.Migracion) error
	ExistByNombre(Nombre string) (*entity.Migracion, error)
}

type MigrationRepositoryImpl struct {
	Base BaseRepository[entity.Migracion]
}

func (m MigrationRepositoryImpl) Save(Migration *entity.Migracion) error {
	return m.Base.Save(Migration)
}

func (m MigrationRepositoryImpl) ExistByNombre(Nombre string) (*entity.Migracion, error) {
	var migration entity.Migracion

	result := m.Base.DB.Where("nombre = ?", Nombre).First(&migration)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, result.Error
	}

	return &migration, nil
}

func NewMigrationRepository(db *gorm.DB) MigrationRepository {
	return &MigrationRepositoryImpl{
		Base: BaseRepository[entity.Migracion]{DB: db},
	}
}
