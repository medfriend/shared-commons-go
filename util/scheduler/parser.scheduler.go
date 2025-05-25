package scheduler

import (
	"encoding/json"
	"fmt"
)

// DataParser interface para parsear los datos JSON a estructuras concretas
type DataParser interface {
	ParseDatosAccion(rawData json.RawMessage, tipoAccion TipoAccion) (DatosAccion, error)
}

// DefaultDataParser implementación del parser
type DefaultDataParser struct{}

func (p *DefaultDataParser) ParseDatosAccion(rawData json.RawMessage, tipoAccion TipoAccion) (DatosAccion, error) {
	switch tipoAccion {
	case TipoAccionHTTP:
		var datos DatosAccionHTTP
		if err := json.Unmarshal(rawData, &datos); err != nil {
			return nil, fmt.Errorf("error parseando datos HTTP: %w", err)
		}
		return datos, nil

	case TipoAccionEmail:
		var datos DatosAccionEmail
		if err := json.Unmarshal(rawData, &datos); err != nil {
			return nil, fmt.Errorf("error parseando datos Email: %w", err)
		}
		return datos, nil

	case TipoAccionScript:
		var datos DatosAccionScript
		if err := json.Unmarshal(rawData, &datos); err != nil {
			return nil, fmt.Errorf("error parseando datos Script: %w", err)
		}
		return datos, nil

	case TipoAccionService:
		var datos DatosAccionService
		if err := json.Unmarshal(rawData, &datos); err != nil {
			return nil, fmt.Errorf("error parseando datos Service: %w", err)
		}
		return datos, nil

	default:
		return nil, fmt.Errorf("tipo de acción no soportado: %s", tipoAccion)
	}
}
