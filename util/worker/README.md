# Worker Pool para servicios Go

Este paquete implementa un sistema de procesamiento concurrente para mejorar el rendimiento de los servicios de Medfriend.

## Características

- **Pool de workers**: Gestión eficiente de goroutines basada en el número de CPUs disponibles
- **Cola de trabajos**: Control de carga con capacidad de rechazo cuando hay sobrecarga
- **Timeouts**: Gestión de timeouts separados para middleware y procesamiento en segundo plano
- **Middleware Gin**: Integración lista para usar con servidores Gin
- **Procesamiento asíncrono**: Capacidad para operaciones "fire and forget" como trazabilidad
- **Monitoreo**: Métricas en tiempo real sobre el estado de los workers
- **Compatibilidad**: Funciones legacy para mantener compatibilidad con código existente

## Uso básico

```go
import (
    "github.com/medfriend/shared-commons-go/util/worker"
    "github.com/gin-gonic/gin"
    "runtime"
)

func main() {
    // Crear un pool con workers = núm. de CPUs y cola de 100 trabajos
    workerPool := worker.NewWorkerPool(runtime.NumCPU(), 100)
    
    // Configurarlo como pool global para middleware
    worker.SetGlobalWorkerPool(workerPool)
    
    // Crear router Gin
    r := gin.Default()
    
    // Aplicar middleware de trazabilidad a todas las rutas
    r.Use(worker.TrazabilidadMiddleware(workerPool))
    
    // Aplicar middleware de workers solo a rutas específicas
    apiGroup := r.Group("/api")
    apiGroup.Use(worker.WorkerPoolMiddleware(
        workerPool,   // pool a usar
        5000,         // timeout middleware (ms)
        30000,        // timeout procesamiento (ms)
        []string{"/health"}, // rutas a excluir
    ))
    
    // Endpoint para métricas
    r.GET("/metrics", worker.MetricsEndpoint(workerPool))
    
    // Al finalizar, cerrar correctamente el pool
    defer workerPool.Shutdown()
    
    r.Run(":8080")
}
```

## Procesamiento en segundo plano

Para operaciones que no necesitan esperar una respuesta (como logging, trazabilidad):

```go
// Crear una solicitud HTTP para el trabajo en segundo plano
req, _ := http.NewRequest("POST", "/internal/procesar-datos", datosBytes)

// Añadir información relevante en headers
req.Header.Set("X-Job-Type", "analisis-datos")
req.Header.Set("X-User-ID", "12345")

// Enviar como trabajo en segundo plano (timeout de 60s)
workerPool.SubmitBackgroundJob(req, 60000)
```

## Solicitudes síncronas

Para operaciones donde se necesita una respuesta:

```go
// Enviar solicitud y obtener canal de respuesta
responseChan := workerPool.SubmitJob(request, 10000)

// Esperar respuesta con timeout
select {
case response := <-responseChan:
    // Procesar respuesta...
case <-time.After(5 * time.Second):
    // Manejar timeout...
}
```

## Métricas y monitoreo

Para obtener estadísticas del pool de workers:

```go
stats := workerPool.GetWorkerStats()
fmt.Printf("Workers: %d total, %d ocupados\n", 
    stats["workers"].(map[string]interface{})["total"],
    stats["workers"].(map[string]interface{})["busy"])
```

También se proporciona un endpoint de Gin listo para usar:

```go
r.GET("/metrics", worker.MetricsEndpoint(workerPool))
```

## Migración desde implementación anterior

Se mantienen las funciones originales por compatibilidad:

```go
// Código anterior
taskQueue := make(chan *http.Request, 100)
stop := make(chan struct{})
worker.CreateWorkers(numCPUs, stop, taskQueue)
```

## Ajustes recomendados

- **Número de workers**: Generalmente igual al número de CPUs (o ligeramente mayor para cargas I/O bound)
- **Tamaño de cola**: Basado en el tráfico esperado (100-500 para servidores medianos)
- **Timeout de middleware**: 3-5 segundos para mantener buena experiencia de usuario
- **Timeout de procesamiento**: 30-60 segundos para operaciones en segundo plano