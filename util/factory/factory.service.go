package factory

import (
    "fmt"
    scdto "github.com/medfriend/shared-commons-go/dto"
)

type FactoryService interface {
	CreateTable(input CreateTableDTO) error
	Query(table string, q QueryDTO) (scdto.PaginatedResponse, error)
	Update(table string, upd UpdateDTO) (int64, error)
	DropTable(table string) error
	CreateRelation(rel RelationDTO) error
	Insert(table string, data InsertDTO) (uint64, error)
	Delete(table string, q DeleteDTO) (int64, error)
}

type FactoryServiceImpl struct {
	repo FactoryRepository
}

func NewFactoryService(repo FactoryRepository) FactoryService {
	return &FactoryServiceImpl{repo: repo}
}

func (s *FactoryServiceImpl) CreateTable(input CreateTableDTO) error {
	return s.repo.CreateTable(input)
}

func (s *FactoryServiceImpl) Query(table string, q QueryDTO) (scdto.PaginatedResponse, error) {
	return s.repo.Query(table, q)
}

func (s *FactoryServiceImpl) Update(table string, upd UpdateDTO) (int64, error) {
    fmt.Println("[Service][Update] table:", table)
    fmt.Println("[Service][Update] payload:", upd)
    rows, err := s.repo.Update(table, upd)
    fmt.Println("[Service][Update] repo response rows:", rows, "err:", err)
    return rows, err
}

func (s *FactoryServiceImpl) DropTable(table string) error {
	return s.repo.DropTable(table)
}

func (s *FactoryServiceImpl) CreateRelation(rel RelationDTO) error {
	return s.repo.CreateRelation(rel)
}

func (s *FactoryServiceImpl) Insert(table string, data InsertDTO) (uint64, error) {
	return s.repo.Insert(table, data.Data)
}

func (s *FactoryServiceImpl) Delete(table string, q DeleteDTO) (int64, error) {
	return s.repo.Delete(table, q)
}
