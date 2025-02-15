package repository

import (
	"github.com/medfriend/shared-commons-go/util/repository"
	"gorm.io/gorm"
	"pay-go/entity"
)

type CardRepository interface {
	Save(Card *entity.Card) error
	FindById(id uint) (*entity.Card, error)
	Find() ([]entity.Card, error)
	Update(Entity *entity.Card) error
	Delete(id uint) error
}

type CardRepositoryImpl struct {
	Base repository.BaseRepository[entity.Card]
}

func NewCardRepository(db *gorm.DB) CardRepository {
	return &CardRepositoryImpl{
		Base: repository.BaseRepository[entity.Card]{DB: db},
	}
}

func (u *CardRepositoryImpl) Save(Card *entity.Card) error {
	return u.Base.Save(Card)
}

func (u *CardRepositoryImpl) FindById(id uint) (*entity.Card, error) {
	return u.Base.FindById(id)
}

func (u *CardRepositoryImpl) Find() ([]entity.Card, error) {
	return u.Base.Find()
}

func (u *CardRepositoryImpl) Update(Card *entity.Card) error {
	return u.Base.Update(Card)
}

func (u *CardRepositoryImpl) Delete(id uint) error {
	return u.Base.Delete(id)
}

