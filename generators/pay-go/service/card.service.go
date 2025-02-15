package service

import (
	"pay-go/entity"
	"pay-go/repository"
)

type CardService interface {
	CreateCard(Card *entity.Card) error
	GetCardById(id uint) (*entity.Card, error)
	GetAllCards() ([]entity.Card, error)
	UpdateCard(Card *entity.Card) error
	DeleteCard(id uint) error
}

type CardServiceImpl struct {
	CardRepo repository.CardRepository
}

func NewCardService(CardRepo repository.CardRepository) CardService {
	return &CardServiceImpl{
		CardRepo: CardRepo,
	}
}

func (s *CardServiceImpl) CreateCard(Card *entity.Card) error {
	return s.CardRepo.Save(Card)
}

func (s *CardServiceImpl) GetCardById(id uint) (*entity.Card, error) {
	return s.CardRepo.FindById(id)
}

func (s *CardServiceImpl) UpdateCard(Card *entity.Card) error {
	return s.CardRepo.Update(Card)
}

func (s *CardServiceImpl) DeleteCard(id uint) error {
	return s.CardRepo.Delete(id)
}

func (s *CardServiceImpl) GetAllCards() ([]entity.Card, error) { return s.CardRepo.Find() }

