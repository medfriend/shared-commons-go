package template

import (
	"fmt"
	"github.com/medfriend/shared-commons-go/generators/util"
)

func GetRepositoryTemplate(args []string) string {

	capitalized := util.CapitalizeFirst(args[0])

	return fmt.Sprintf(`package repository

import (
	"github.com/medfriend/shared-commons-go/util/repository"
	"gorm.io/gorm"
	"%s-go/entity"
)

type %sRepository interface {
	Save(%s *entity.%s) error
	FindById(id uint) (*entity.%s, error)
	Find() ([]entity.%s, error)
	Update(Entity *entity.%s) error
	Delete(id uint) error
}

type %sRepositoryImpl struct {
	Base repository.BaseRepository[entity.%s]
}

func New%sRepository(db *gorm.DB) %sRepository {
	return &%sRepositoryImpl{
		Base: repository.BaseRepository[entity.%s]{DB: db},
	}
}

func (u *%sRepositoryImpl) Save(%s *entity.%s) error {
	return u.Base.Save(%s)
}

func (u *%sRepositoryImpl) FindById(id uint) (*entity.%s, error) {
	return u.Base.FindById(id)
}

func (u *%sRepositoryImpl) Find() ([]entity.%s, error) {
	return u.Base.Find()
}

func (u *%sRepositoryImpl) Update(%s *entity.%s) error {
	return u.Base.Update(%s)
}

func (u *%sRepositoryImpl) Delete(id uint) error {
	return u.Base.Delete(id)
}

`, args[1],
		capitalized, capitalized, capitalized, capitalized, capitalized, capitalized, capitalized,
		capitalized, capitalized, capitalized, capitalized, capitalized, capitalized, capitalized,
		capitalized, capitalized, capitalized, capitalized, capitalized, capitalized, capitalized,
		capitalized, capitalized, capitalized, capitalized)
}
