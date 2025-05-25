package scheduler

import (
	"encoding/json"
	"fmt"
	"time"
)

// TareaProgramada representa el modelo de la tabla tareas_programadas
type TareaProgramada struct {
	ID                    int64           `db:"tareas_programadas_id" json:"id"`
	Nombre                string          `db:"nombre" json:"nombre"`
	Descripcion           string          `db:"descripcion" json:"descripcion"`
	ExpresionProgramacion string          `db:"expresion_programacion" json:"expresion_programacion"`
	TipoAccion            string          `db:"tipo_accion" json:"tipo_accion"`
	DatosAccion           json.RawMessage `db:"datos_accion" json:"datos_accion"`
	Activo                bool            `db:"activo" json:"activo"`
	FechaCreacion         time.Time       `db:"fecha_creacion" json:"fecha_creacion"`
	FechaActualizacion    *time.Time      `db:"fecha_actualizacion" json:"fecha_actualizacion"`
	TiempoMaximo          *float64        `db:"tiempo_maximo" json:"tiempo_maximo"`
	PoliticaReintentos    json.RawMessage `db:"politica_reintentos" json:"politica_reintentos"`
	ServicioID            *int            `db:"md_service_servicio_id" json:"servicio_id"`
}

// PoliticaReintentos define cómo se deben manejar los reintentos
type PoliticaReintentos struct {
	MaxReintentos      int           `json:"max_reintentos"`
	IntervaloReintento time.Duration `json:"intervalo_reintento"`
	BackoffMultiplier  float64       `json:"backoff_multiplier"`
	MaxIntervalo       time.Duration `json:"max_intervalo"`
}

// DatosAccion es la interface base para todos los tipos de datos de acción
type DatosAccion interface {
	Validar() error
}

// DatosAccionHTTP para llamadas HTTP
type DatosAccionHTTP struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    json.RawMessage   `json:"body"`
}

func (d DatosAccionHTTP) Validar() error {
	// Validación específica para HTTP
	return nil
}

// DatosAccionEmail para envío de emails
type DatosAccionEmail struct {
	Para     []string `json:"para"`
	Asunto   string   `json:"asunto"`
	Cuerpo   string   `json:"cuerpo"`
	Adjuntos []string `json:"adjuntos"`
}

func (d DatosAccionEmail) Validar() error {
	// Validación específica para Email
	return nil
}

// DatosAccionScript para ejecutar scripts
type DatosAccionScript struct {
	Comando    string   `json:"comando"`
	Argumentos []string `json:"argumentos"`
	Directorio string   `json:"directorio"`
}

func (d DatosAccionScript) Validar() error {
	// Validación específica para Scripts
	return nil
}

// DatosAccionService para ejecutar funciones de servicios internos
type DatosAccionService struct {
	Servicio   string                 `json:"servicio"`
	Funcion    string                 `json:"funcion"`
	Parametros map[string]interface{} `json:"parametros"`
}

func (d DatosAccionService) Validar() error {
	// Validación específica para Services
	if d.Servicio == "" {
		return fmt.Errorf("servicio es requerido")
	}
	if d.Funcion == "" {
		return fmt.Errorf("funcion es requerida")
	}
	return nil
}

// TipoAccion define los tipos de acciones disponibles
type TipoAccion string

const (
	TipoAccionHTTP    TipoAccion = "HTTP"
	TipoAccionEmail   TipoAccion = "EMAIL"
	TipoAccionScript  TipoAccion = "SCRIPT"
	TipoAccionDB      TipoAccion = "DATABASE"
	TipoAccionService TipoAccion = "SERVICE"
)

// EstadoEjecucion representa el estado de una ejecución
type EstadoEjecucion string

const (
	EstadoPendiente  EstadoEjecucion = "PENDIENTE"
	EstadoEjecutando EstadoEjecucion = "EJECUTANDO"
	EstadoCompletado EstadoEjecucion = "COMPLETADO"
	EstadoFallido    EstadoEjecucion = "FALLIDO"
	EstadoCancelado  EstadoEjecucion = "CANCELADO"
)

// ResultadoEjecucion almacena el resultado de ejecutar una tarea
type ResultadoEjecucion struct {
	TareaID          int64           `json:"tarea_id"`
	FechaInicio      time.Time       `json:"fecha_inicio"`
	FechaFin         *time.Time      `json:"fecha_fin"`
	Estado           EstadoEjecucion `json:"estado"`
	Resultado        json.RawMessage `json:"resultado"`
	Error            string          `json:"error,omitempty"`
	IntentoNumero    int             `json:"intento_numero"`
	DuracionSegundos float64         `json:"duracion_segundos"`
}
