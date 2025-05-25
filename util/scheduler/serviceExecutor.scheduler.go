package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"
)

// Service interfaz genérica para cualquier servicio
type Service interface{}

// ServiceRegistry mapa de servicios registrados
type ServiceRegistry map[string]Service

// ServiceContainer contiene todos los servicios disponibles
type ServiceContainer struct {
	services ServiceRegistry
}

// NewServiceContainer crea un nuevo contenedor de servicios
func NewServiceContainer() *ServiceContainer {
	return &ServiceContainer{
		services: make(ServiceRegistry),
	}
}

// RegisterService registra un servicio en el contenedor
func (sc *ServiceContainer) RegisterService(name string, service Service) {
	sc.services[name] = service
}

// GetRegisteredServices retorna todos los servicios registrados
func (sc *ServiceContainer) GetRegisteredServices() ServiceRegistry {
	return sc.services
}

// ServiceExecutor ejecutor para tareas de servicios internos
type ServiceExecutor struct {
	BaseExecutor
	serviceContainer *ServiceContainer
}

// NewServiceExecutor crea un nuevo ejecutor de servicios
func NewServiceExecutor(parser DataParser, serviceContainer *ServiceContainer) TaskExecutor {
	return &ServiceExecutor{
		BaseExecutor:     NewBaseExecutor(TipoAccionService, parser),
		serviceContainer: serviceContainer,
	}
}

// Execute implementa la ejecución de una tarea de servicio
func (e *ServiceExecutor) Execute(ctx context.Context, tarea *TareaProgramada) (*ResultadoEjecucion, error) {
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

	datosService, ok := datos.(DatosAccionService)
	if !ok {
		err := fmt.Errorf("tipo de datos incorrecto para SERVICE")
		result.Estado = EstadoFallido
		result.Error = err.Error()
		result.FechaFin = &time.Time{}
		*result.FechaFin = time.Now()
		return result, err
	}

	// Obtener el servicio
	serviceValue, err := e.getService(datosService.Servicio)
	if err != nil {
		result.Estado = EstadoFallido
		result.Error = err.Error()
		result.FechaFin = &time.Time{}
		*result.FechaFin = time.Now()
		return result, err
	}

	// Ejecutar la función del servicio
	resultData, err := e.executeServiceFunction(serviceValue, datosService.Funcion, datosService.Parametros)
	if err != nil {
		result.Estado = EstadoFallido
		result.Error = err.Error()
		result.FechaFin = &time.Time{}
		*result.FechaFin = time.Now()
		return result, err
	}

	// Serializar el resultado
	if resultData != nil {
		resultBytes, err := json.Marshal(resultData)
		if err != nil {
			result.Estado = EstadoFallido
			result.Error = fmt.Sprintf("error al serializar resultado: %v", err)
		} else {
			result.Resultado = resultBytes
			result.Estado = EstadoCompletado
		}
	} else {
		result.Estado = EstadoCompletado
	}

	result.FechaFin = &time.Time{}
	*result.FechaFin = time.Now()

	return result, nil
}

// getService obtiene el servicio por nombre
func (e *ServiceExecutor) getService(serviceName string) (reflect.Value, error) {
	service, exists := e.serviceContainer.services[serviceName]
	if !exists {
		return reflect.Value{}, fmt.Errorf("servicio '%s' no encontrado", serviceName)
	}

	serviceValue := reflect.ValueOf(service)
	if !serviceValue.IsValid() || (serviceValue.Kind() == reflect.Ptr && serviceValue.IsNil()) {
		return reflect.Value{}, fmt.Errorf("servicio '%s' no está inicializado", serviceName)
	}

	return serviceValue, nil
}

// executeServiceFunction ejecuta una función específica del servicio usando reflection
func (e *ServiceExecutor) executeServiceFunction(serviceValue reflect.Value, functionName string, params map[string]interface{}) (interface{}, error) {
	method := serviceValue.MethodByName(functionName)
	if !method.IsValid() {
		return nil, fmt.Errorf("método '%s' no encontrado en el servicio", functionName)
	}

	methodType := method.Type()
	numParams := methodType.NumIn()

	// Preparar argumentos
	args := make([]reflect.Value, numParams)

	// Si hay parámetros, intentar convertirlos
	if numParams > 0 && params != nil {
		for i := 0; i < numParams; i++ {
			paramType := methodType.In(i)

			// Buscar el parámetro por índice o por nombre aproximado
			var paramValue interface{}
			var found bool

			// Intentar obtener el parámetro por índice
			if paramKey := fmt.Sprintf("param%d", i); params[paramKey] != nil {
				paramValue = params[paramKey]
				found = true
			} else {
				// Si solo hay un parámetro en el map, usarlo
				if len(params) == 1 {
					for _, v := range params {
						paramValue = v
						found = true
						break
					}
				}
			}

			if found {
				// Convertir el parámetro al tipo esperado
				convertedValue, err := e.convertToType(paramValue, paramType)
				if err != nil {
					return nil, fmt.Errorf("error convirtiendo parámetro %d: %v", i, err)
				}
				args[i] = convertedValue
			} else {
				// Si no se encuentra el parámetro, usar valor cero
				args[i] = reflect.Zero(paramType)
			}
		}
	}

	// Ejecutar el método
	results := method.Call(args)

	// Procesar resultados
	if len(results) == 0 {
		return nil, nil
	}

	// Si el último valor de retorno es un error, verificarlo
	if len(results) > 1 {
		lastResult := results[len(results)-1]
		if lastResult.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			if !lastResult.IsNil() {
				return nil, lastResult.Interface().(error)
			}
		}
	}

	// Retornar el primer resultado (si no es error)
	if len(results) == 1 {
		if results[0].Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			if !results[0].IsNil() {
				return nil, results[0].Interface().(error)
			}
			return nil, nil
		}
		return results[0].Interface(), nil
	}

	// Si hay múltiples resultados, retornar el primero (excluyendo el error)
	return results[0].Interface(), nil
}

// convertToType convierte un valor a un tipo específico
func (e *ServiceExecutor) convertToType(value interface{}, targetType reflect.Type) (reflect.Value, error) {
	if value == nil {
		return reflect.Zero(targetType), nil
	}

	sourceValue := reflect.ValueOf(value)

	// Si los tipos son compatibles, convertir directamente
	if sourceValue.Type().ConvertibleTo(targetType) {
		return sourceValue.Convert(targetType), nil
	}

	// Si el tipo objetivo es un puntero, crear uno
	if targetType.Kind() == reflect.Ptr {
		elemType := targetType.Elem()
		convertedElem, err := e.convertToType(value, elemType)
		if err != nil {
			return reflect.Value{}, err
		}
		ptr := reflect.New(elemType)
		ptr.Elem().Set(convertedElem)
		return ptr, nil
	}

	// Si es una estructura, intentar deserializar desde JSON
	if targetType.Kind() == reflect.Struct {
		jsonBytes, err := json.Marshal(value)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("error convirtiendo a JSON: %v", err)
		}

		newValue := reflect.New(targetType)
		err = json.Unmarshal(jsonBytes, newValue.Interface())
		if err != nil {
			return reflect.Value{}, fmt.Errorf("error deserializando JSON: %v", err)
		}

		return newValue.Elem(), nil
	}

	return reflect.Zero(targetType), fmt.Errorf("no se puede convertir %T a %v", value, targetType)
}

// ValidarDatos valida los datos específicos de servicios
func (e *ServiceExecutor) ValidarDatos(datosAccion DatosAccion) error {
	datosService, ok := datosAccion.(DatosAccionService)
	if !ok {
		return fmt.Errorf("datos no son del tipo SERVICE")
	}

	return datosService.Validar()
}
