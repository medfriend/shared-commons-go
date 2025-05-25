package scheduler

import (
	"encoding/json"
	"time"
)

// BaseExecutor proporciona funcionalidad común para todos los ejecutores
type BaseExecutor struct {
	tipo   TipoAccion
	parser DataParser
}

// NewBaseExecutor crea una nueva instancia del executor base
func NewBaseExecutor(tipo TipoAccion, parser DataParser) BaseExecutor {
	return BaseExecutor{
		tipo:   tipo,
		parser: parser,
	}
}

// GetTipo retorna el tipo de acción que maneja este ejecutor
func (b *BaseExecutor) GetTipo() TipoAccion {
	return b.tipo
}

// ParseData helper method para parsear datos
func (b *BaseExecutor) ParseData(rawData json.RawMessage) (DatosAccion, error) {
	return b.parser.ParseDatosAccion(rawData, b.tipo)
}

// CreateResult crea un resultado de ejecución con valores por defecto
func (b *BaseExecutor) CreateResult(tareaID int64) *ResultadoEjecucion {
	return &ResultadoEjecucion{
		TareaID:       tareaID,
		FechaInicio:   time.Now(),
		Estado:        EstadoEjecutando,
		IntentoNumero: 1,
	}
}
