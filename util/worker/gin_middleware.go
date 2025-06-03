package worker

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/medfriend/shared-commons-go/util/logger"
	"go.uber.org/zap"
	"io"
	"net/http"
	"time"
)

var globalWorkerPool *WorkerPool

// SetGlobalWorkerPool configura el pool de workers para los middleware
func SetGlobalWorkerPool(pool *WorkerPool) {
	globalWorkerPool = pool
}

// GetGlobalWorkerPool retorna el pool de workers global configurado
// Retorna nil si no hay ningún pool configurado
func GetGlobalWorkerPool() *WorkerPool {
	return globalWorkerPool
}

// WorkerPoolMiddleware distribuye las solicitudes entrantes al pool de workers
// Parámetros:
// - pool: el pool de workers a utilizar (si es nil, usará el global)
// - timeoutMillis: timeout en milisegundos para el middleware (si es 0, usará 5000ms)
// - processorTimeout: timeout en milisegundos para el procesador (si es 0, usará 30000ms)
// - excludePaths: paths a excluir del procesamiento async (ej. "/health")
func WorkerPoolMiddleware(pool *WorkerPool, timeoutMillis int64, processorTimeout int64, excludePaths []string) gin.HandlerFunc {
	// Usar valores por defecto si no se proporcionan
	if pool == nil {
		pool = globalWorkerPool
		if pool == nil {
			// Crear un pool por defecto si no existe
			logger.GetLogger().Warn("Creando WorkerPool por defecto en el middleware")
			pool = NewWorkerPool(0, 100) // 0 = usar número de CPUs
			globalWorkerPool = pool
		}
	}
	
	if timeoutMillis <= 0 {
		timeoutMillis = 5000 // 5 segundos
	}
	
	if processorTimeout <= 0 {
		processorTimeout = 30000 // 30 segundos
	}
	
	// Mapa para búsqueda eficiente de paths excluidos
	excludeMap := make(map[string]bool)
	for _, path := range excludePaths {
		excludeMap[path] = true
	}

	return func(c *gin.Context) {
		// Verificar si el path está excluido
		if excludeMap[c.Request.URL.Path] {
			c.Next()
			return
		}
		
		// También excluir endpoints comunes de salud/métricas por defecto
		if c.Request.URL.Path == "/health" || c.Request.URL.Path == "/metrics" || 
		   c.Request.URL.Path == "/readiness" || c.Request.URL.Path == "/liveness" {
			c.Next()
			return
		}

		// Duplicar el cuerpo de la solicitud para poder leerlo múltiples veces
		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
			// Restaurar el cuerpo para que pueda ser leído de nuevo
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// Crear una copia de la solicitud
		reqCopy := &http.Request{
			Method:        c.Request.Method,
			URL:           c.Request.URL,
			Header:        c.Request.Header.Clone(),
			Body:          io.NopCloser(bytes.NewBuffer(bodyBytes)),
			ContentLength: c.Request.ContentLength,
			Host:          c.Request.Host,
			RemoteAddr:    c.Request.RemoteAddr,
		}

		// Enviar la solicitud al pool de workers
		responseChan := pool.SubmitJob(reqCopy, processorTimeout)

		// Esperar la respuesta del worker o el timeout
		timeoutChan := time.After(time.Duration(timeoutMillis) * time.Millisecond)
		
		select {
		case workerResp, ok := <-responseChan:
			if ok && workerResp != nil {
				// En una implementación real, transferirías los datos de la respuesta
				// a la respuesta de Gin. Por ahora, solo registramos el resultado.
				logger.GetLogger().Info("Respuesta recibida del worker",
					zap.String("path", c.Request.URL.Path),
					zap.String("status", workerResp.Status))
			}
		case <-timeoutChan:
			// El timeout del middleware es más corto que el del worker, lo que permite
			// que los workers continúen trabajando en segundo plano mientras liberamos
			// el hilo de procesamiento HTTP
			logger.GetLogger().Info("Middleware avanzando sin esperar al worker",
				zap.String("path", c.Request.URL.Path),
				zap.String("note", "el worker sigue procesando en segundo plano"))
		}

		// Continuar con la cadena de middleware normal
		c.Next()
	}
}

// TrazabilidadMiddleware middleware para registrar trazabilidad en segundo plano
func TrazabilidadMiddleware(pool *WorkerPool) gin.HandlerFunc {
	// Usar el pool global si no se proporciona uno
	if pool == nil {
		pool = globalWorkerPool
		if pool == nil {
			logger.GetLogger().Warn("TrazabilidadMiddleware requiere un WorkerPool configurado")
			return func(c *gin.Context) { c.Next() } // Middleware nulo si no hay pool
		}
	}

	return func(c *gin.Context) {
		// Capturar información relevante antes de c.Next()
		requestURL := c.Request.URL.Path
		requestMethod := c.Request.Method
		authHeader := c.Request.Header.Get("Authorization")
		clientIP := c.ClientIP()
		startTime := time.Now()
		
		// Procesar la solicitud normalmente
		c.Next()
		
		// Capturar información adicional después del procesamiento
		statusCode := c.Writer.Status()
		latency := time.Since(startTime)
		
		// Solo registrar trazas si hay una autorización (usuario autenticado)
		if authHeader != "" {
			// Crear una request para la trazabilidad
			trazaReq, _ := http.NewRequest("POST", "/internal/trazabilidad", nil)
			
			// Añadir información relevante en headers para procesar después
			trazaReq.Header.Set("X-Original-URL", requestURL)
			trazaReq.Header.Set("X-Original-Method", requestMethod)
			trazaReq.Header.Set("Authorization", authHeader)
			trazaReq.Header.Set("X-Response-Status", fmt.Sprintf("%d", statusCode))
			trazaReq.Header.Set("X-Client-IP", clientIP)
			trazaReq.Header.Set("X-Latency-Ms", fmt.Sprintf("%d", latency.Milliseconds()))
			
			// Enviar como trabajo en segundo plano con un timeout largo (30s)
			pool.SubmitBackgroundJob(trazaReq, 30000)
		}
	}
}

// LoggingMiddleware registra información detallada sobre la solicitud y respuesta
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Tiempo de inicio
		startTime := time.Now()

		// Procesar solicitud
		c.Next()

		// Tiempo de finalización
		endTime := time.Now()
		latency := endTime.Sub(startTime)

		// Registrar detalles
		logger.GetLogger().Info("HTTP Request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("client_ip", c.ClientIP()),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
		)
	}
}

// MetricsEndpoint crea un handler de Gin para exponer métricas del worker pool
func MetricsEndpoint(pool *WorkerPool) gin.HandlerFunc {
	if pool == nil {
		pool = globalWorkerPool
	}
	
	return func(c *gin.Context) {
		if pool == nil {
			c.JSON(http.StatusOK, gin.H{
				"status": "no worker pool configured",
			})
			return
		}
		
		stats := pool.GetWorkerStats()
		stats["server"] = map[string]interface{}{
			"uptime": time.Now().Format(time.RFC3339),
		}
		
		c.JSON(http.StatusOK, stats)
	}
}