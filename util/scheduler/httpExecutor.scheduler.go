package scheduler

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// HTTPExecutor ejecutor para tareas HTTP
type HTTPExecutor struct {
	BaseExecutor
	httpClient *http.Client
}

// NewHTTPExecutor crea un nuevo ejecutor HTTP
func NewHTTPExecutor(parser DataParser, httpClient *http.Client) TaskExecutor {
	return &HTTPExecutor{
		BaseExecutor: NewBaseExecutor(TipoAccionHTTP, parser),
		httpClient:   httpClient,
	}
}

// Execute implementa la ejecución de una tarea HTTP
func (e *HTTPExecutor) Execute(ctx context.Context, tarea *TareaProgramada) (*ResultadoEjecucion, error) {
	result := e.CreateResult(tarea.ID)

	// Parse datos
	datos, err := e.ParseData(tarea.DatosAccion)
	if err != nil {
		result.Estado = EstadoFallido
		result.Error = err.Error()
		result.FechaFin = &time.Time{}
		*result.FechaFin = time.Now()
		return result, err
	}

	datosHTTP, ok := datos.(DatosAccionHTTP)
	if !ok {
		err := fmt.Errorf("tipo de datos incorrecto para HTTP")
		result.Estado = EstadoFallido
		result.Error = err.Error()
		result.FechaFin = &time.Time{}
		*result.FechaFin = time.Now()
		return result, err
	}

	// Crear request HTTP
	req, err := http.NewRequestWithContext(ctx, datosHTTP.Method, datosHTTP.URL, nil)
	if err != nil {
		result.Estado = EstadoFallido
		result.Error = err.Error()
		result.FechaFin = &time.Time{}
		*result.FechaFin = time.Now()
		return result, err
	}

	// Agregar headers
	for key, value := range datosHTTP.Headers {
		req.Header.Set(key, value)
	}

	// Ejecutar request
	resp, err := e.httpClient.Do(req)
	if err != nil {
		result.Estado = EstadoFallido
		result.Error = err.Error()
		result.FechaFin = &time.Time{}
		*result.FechaFin = time.Now()
		return result, err
	}
	defer resp.Body.Close()

	// Verificar status
	if resp.StatusCode >= 400 {
		result.Estado = EstadoFallido
		result.Error = fmt.Sprintf("HTTP status code: %d", resp.StatusCode)
	} else {
		result.Estado = EstadoCompletado
	}

	result.FechaFin = &time.Time{}
	*result.FechaFin = time.Now()

	return result, nil
}

// ValidarDatos valida los datos específicos de HTTP
func (e *HTTPExecutor) ValidarDatos(datosAccion DatosAccion) error {
	datosHTTP, ok := datosAccion.(DatosAccionHTTP)
	if !ok {
		return fmt.Errorf("datos no son del tipo HTTP")
	}

	return datosHTTP.Validar()
}
