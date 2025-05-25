package scheduler

import (
	"context"
	"fmt"
	"sync"
)

// DefaultTaskFactory implementación concreta del factory
type DefaultTaskFactory struct {
	executors map[TipoAccion]TaskExecutor
	mu        sync.RWMutex
}

// TaskExecutor define la interface que deben implementar todos los ejecutores
type TaskExecutor interface {
	Execute(ctx context.Context, tarea *TareaProgramada) (*ResultadoEjecucion, error)
	ValidarDatos(datosAccion DatosAccion) error
	GetTipo() TipoAccion
}

// TaskFactory es la interface principal del factory para crear ejecutores
type TaskFactory interface {
	CreateExecutor(tipoAccion TipoAccion) (TaskExecutor, error)
	RegisterExecutor(tipoAccion TipoAccion, executor TaskExecutor)
	GetExecutorTypes() []TipoAccion
}

// CreateExecutor crea un ejecutor basado en el tipo de acción
func (f *DefaultTaskFactory) CreateExecutor(tipoAccion TipoAccion) (TaskExecutor, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	executor, exists := f.executors[tipoAccion]
	if !exists {
		return nil, fmt.Errorf("ejecutor no encontrado para tipo: %s", tipoAccion)
	}

	return executor, nil
}

// RegisterExecutor registra un nuevo ejecutor en el factory
func (f *DefaultTaskFactory) RegisterExecutor(tipoAccion TipoAccion, executor TaskExecutor) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.executors[tipoAccion] = executor
}

// GetExecutorTypes retorna todos los tipos de ejecutores disponibles
func (f *DefaultTaskFactory) GetExecutorTypes() []TipoAccion {
	f.mu.RLock()
	defer f.mu.RUnlock()

	types := make([]TipoAccion, 0, len(f.executors))
	for tipoAccion := range f.executors {
		types = append(types, tipoAccion)
	}

	return types
}

// NewTaskFactory crea una nueva instancia del factory
func NewTaskFactory() TaskFactory {
	return &DefaultTaskFactory{
		executors: make(map[TipoAccion]TaskExecutor),
	}
}
