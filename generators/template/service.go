package template

import (
	"fmt"
	"github.com/medfriend/shared-commons-go/generators/util"
)

func GetService(args []string) string {
	capitalized := util.CapitalizeFirst(args[0])

	return fmt.Sprintf(`package service

import (
	"%s-go/entity"
	"%s-go/repository"
)

type %sService interface {
	Create%s(%s *entity.%s) error
	Get%sById(id uint) (*entity.%s, error)
	GetAll%ss() ([]entity.%s, error)
	Update%s(%s *entity.%s) error
	Delete%s(id uint) error
}

type %sServiceImpl struct {
	%sRepo repository.%sRepository
}

func New%sService(%sRepo repository.%sRepository) %sService {
	return &%sServiceImpl{
		%sRepo: %sRepo,
	}
}

func (s *%sServiceImpl) Create%s(%s *entity.%s) error {
	return s.%sRepo.Save(%s)
}

func (s *%sServiceImpl) Get%sById(id uint) (*entity.%s, error) {
	return s.%sRepo.FindById(id)
}

func (s *%sServiceImpl) Update%s(%s *entity.%s) error {
	return s.%sRepo.Update(%s)
}

func (s *%sServiceImpl) Delete%s(id uint) error {
	return s.%sRepo.Delete(id)
}

func (s *%sServiceImpl) GetAll%ss() ([]entity.%s, error) { return s.%sRepo.Find() }

`, args[1], args[1], capitalized, capitalized, capitalized, capitalized, capitalized,
		capitalized, capitalized, capitalized, capitalized, capitalized, capitalized,
		capitalized, capitalized, capitalized, capitalized, capitalized, capitalized,
		capitalized, capitalized, capitalized, capitalized, capitalized, capitalized,
		capitalized, capitalized, capitalized, capitalized, capitalized, capitalized,
		capitalized, capitalized, capitalized, capitalized, capitalized, capitalized,
		capitalized, capitalized, capitalized, capitalized, capitalized, capitalized,
		capitalized, capitalized, capitalized, capitalized)
}
