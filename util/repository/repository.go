package repository

import (
	"fmt"
	"github.com/medfriend/shared-commons-go/dto"
	loggerR "github.com/medfriend/shared-commons-go/util/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"math"
	"reflect"
	"sync"
)

type BaseRepository[T any] struct {
	DB *gorm.DB
}

func (r *BaseRepository[T]) Pagination(paginacion dto.PaginationDTO) (dto.PaginatedResponse, error) {
	return r.PaginationWithQuery(paginacion, nil)
}

// PaginationWithQuery permite aplicar una consulta personalizada además de la paginación
// El parámetro queryFunc recibe un *gorm.DB y debe retornar un *gorm.DB modificado con las condiciones deseadas
func (r *BaseRepository[T]) PaginationWithQuery(paginacion dto.PaginationDTO, queryFunc func(*gorm.DB) *gorm.DB) (dto.PaginatedResponse, error) {
	var entities []T
	var totalRows int64

	// Query base
	query := r.DB.Model(new(T))

	// Aplicar la función de consulta si fue proporcionada
	if queryFunc != nil {
		query = queryFunc(query)
	}

	// Contar registros totales aplicando los filtros
	query.Count(&totalRows)

	// Calcular offset para paginación
	offset := (paginacion.Pagina - 1) * paginacion.Filas

	// Aplicar paginación y ejecutar consulta final
	err := query.Limit(int(paginacion.Filas)).Offset(int(offset)).Find(&entities).Error

	// Calcular páginas totales
	totalPages := int(math.Ceil(float64(totalRows) / float64(paginacion.Filas)))

	paginacionResponse := dto.PaginatedResponse{
		Data:       entities,
		Total:      totalRows,
		Page:       int(paginacion.Pagina),
		PageSize:   int(paginacion.Filas),
		TotalPages: totalPages,
	}

	return paginacionResponse, err
}

func (r *BaseRepository[T]) Save(entity *T) error {
	return r.DB.Create(entity).Error
}

func (r *BaseRepository[T]) FindById(id uint) (*T, error) {
	var entity T
	result := r.DB.First(&entity, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &entity, nil
}

func (r *BaseRepository[T]) Find() ([]T, error) {
	var entities []T
	result := r.DB.Find(&entities)
	if result.Error != nil {
		return nil, result.Error
	}
	return entities, nil
}

func (r *BaseRepository[T]) FindByIdWithRelations(id uint, relations ...string) (*T, error) {
	var entity T

	query := r.DB

	for _, relation := range relations {
		query = query.Preload(relation)
	}

	result := query.First(&entity, id)

	if result.Error != nil {
		return nil, result.Error
	}
	return &entity, nil
}

func (r *BaseRepository[T]) FindAnyField(fields []string, query string, castFields map[string]bool) (*[]T, error) {
	var results []T

	if len(fields) == 0 {
		return nil, fmt.Errorf("no fields provided to search")
	}

	dbQuery := r.DB.Distinct()
	for i, field := range fields {
		fieldExpr := field
		if castFields[field] {
			fieldExpr = fmt.Sprintf("CAST(%s AS TEXT)", field)
		}

		condition := fmt.Sprintf("%s ILIKE ?", fieldExpr)
		if i == 0 {
			dbQuery = dbQuery.Where(condition, "%"+query+"%")
		} else {
			dbQuery = dbQuery.Or(condition, "%"+query+"%")
		}
	}

	if err := dbQuery.Find(&results).Error; err != nil {
		return nil, err
	}

	return &results, nil
}

// La función ahora acepta un mapa `relationFieldMap` como parámetro
func (r *BaseRepository[T]) FindByIdWithRelationsAsync(id uint, relations map[string]string) (*T, error) {
	var entity T
	var wg sync.WaitGroup
	errChan := make(chan error, len(relations)) // Canal para manejar errores en goroutines

	// Cargar la entidad principal sin relaciones
	logger := loggerR.GetLogger()
	logger.Debug("Iniciando carga de la entidad principal", zap.Uint("id", id))
	if err := r.DB.First(&entity, id).Error; err != nil {
		logger.Error("Error al cargar la entidad principal",
			zap.Uint("id", id),
			zap.Error(err))
		return nil, err
	}
	logger.Debug("Entidad principal cargada exitosamente")

	// Obtener un reflejo del valor de la entidad para inspeccionar sus campos
	entityValue := reflect.ValueOf(&entity).Elem()

	// Función para cargar cada relación en paralelo
	loadRelation := func(rel string) {
		defer wg.Done()
		logger := loggerR.GetLogger()
		logger.Debug("Iniciando carga de relación", zap.String("relacion", rel))

		// Cargar solo la relación en `entity`
		if err := r.DB.Model(&entity).Preload(rel).Find(&entity).Error; err != nil {
			logger.Error("Error al cargar relación",
				zap.String("relacion", rel),
				zap.Error(err))
			errChan <- err // Enviar error al canal si ocurre otro tipo de error
			return
		}
		logger.Debug("Relación cargada exitosamente", zap.String("relacion", rel))
	}

	for i := 0; i < entityValue.NumField(); i++ {
		field := entityValue.Type().Field(i)
		value := entityValue.Field(i)
		logger.Debug("Campo de entidad",
			zap.String("campo", field.Name),
			zap.Any("valor", value.Interface()))
	}

	// Ejecutar `Preload` solo para las relaciones que no son `nil` en la entidad
	for relation, fieldName := range relations {
		// Verificar si el campo correspondiente a la relación es `nil`
		field := entityValue.FieldByName(fieldName)
		if field.IsValid() && !field.IsZero() { // Usamos IsZero para verificar si está vacío
			wg.Add(1)
			go loadRelation(relation)
		} else {
			logger.Debug("Relación omitida por ser nil o vacía",
				zap.String("relacion", relation),
				zap.String("campo", fieldName))
		}
	}

	// Esperar a que todas las goroutines terminen
	go func() {
		wg.Wait()
		close(errChan) // Cerrar el canal de errores al terminar
	}()

	// Verificar si hubo errores
	for err := range errChan {
		if err != nil {
			logger.Error("Error encontrado en el canal de errores", zap.Error(err))
			return nil, err // Retornar el primer error encontrado
		}
	}

	logger.Debug("Todas las relaciones cargadas exitosamente")
	return &entity, nil
}

func (r *BaseRepository[T]) Update(entity *T) error {
	return r.DB.Save(entity).Error
}

func (r *BaseRepository[T]) Delete(id uint) error {
	var entity T
	result := r.DB.Delete(&entity, id)
	return result.Error
}
