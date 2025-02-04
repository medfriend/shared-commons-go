package repository

import (
	"fmt"
	"gorm.io/gorm"
	"reflect"
	"sync"
)

type BaseRepository[T any] struct {
	DB *gorm.DB
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

func (r *BaseRepository[T]) FindAnyField(fields []string, query string) (*[]T, error) {
	var results []T

	if len(fields) == 0 {
		return nil, fmt.Errorf("no fields provided to search")
	}

	dbQuery := r.DB
	for i, field := range fields {
		// Usamos `OR` para buscar en múltiples campos
		if i == 0 {
			fmt.Println()
			dbQuery = dbQuery.Where(fmt.Sprintf("%s ILIKE ?", field), "%"+query+"%")
		} else {
			dbQuery = dbQuery.Or(fmt.Sprintf("%s ILIKE ?", field), "%"+query+"%")
		}
		fmt.Println(dbQuery)
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
	fmt.Println("Iniciando carga de la entidad principal...")
	if err := r.DB.First(&entity, id).Error; err != nil {
		fmt.Printf("Error al cargar la entidad principal con ID %d: %v\n", id, err)
		return nil, err
	}
	fmt.Println("Entidad principal cargada exitosamente.")

	// Obtener un reflejo del valor de la entidad para inspeccionar sus campos
	entityValue := reflect.ValueOf(&entity).Elem()

	// Función para cargar cada relación en paralelo
	loadRelation := func(rel string) {
		defer wg.Done()
		fmt.Printf("Iniciando carga de la relación '%s'...\n", rel)

		// Cargar solo la relación en `entity`
		if err := r.DB.Model(&entity).Preload(rel).Find(&entity).Error; err != nil {
			fmt.Printf("Error al cargar la relación '%s': %v\n", rel, err)
			errChan <- err // Enviar error al canal si ocurre otro tipo de error
			return
		}
		fmt.Printf("Relación '%s' cargada exitosamente.\n", rel)
	}

	fmt.Println(entity)
	fmt.Println(entityValue)

	fmt.Println("Campos de entityValue:")
	for i := 0; i < entityValue.NumField(); i++ {
		field := entityValue.Type().Field(i)
		value := entityValue.Field(i)
		fmt.Printf("Campo: %s, Valor: %v\n", field.Name, value)
	}

	// Ejecutar `Preload` solo para las relaciones que no son `nil` en la entidad
	for relation, fieldName := range relations {
		// Verificar si el campo correspondiente a la relación es `nil`
		field := entityValue.FieldByName(fieldName)
		if field.IsValid() && !field.IsZero() { // Usamos IsZero para verificar si está vacío
			wg.Add(1)
			go loadRelation(relation)
		} else {
			fmt.Printf("Relación '%s' es nil o vacía en la entidad (campo: '%s'), se omite el preload.\n", relation, fieldName)
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
			fmt.Println("Error encontrado en el canal de errores:", err)
			return nil, err // Retornar el primer error encontrado
		}
	}

	fmt.Println("Todas las relaciones cargadas exitosamente.")
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
