package httpServer

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/medfriend/shared-commons-go/util/logger"
	"github.com/medfriend/shared-commons-go/util/worker"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"net/http"
	"time"
)

// InitHttpServerWithWorkerPool inicializa el servidor HTTP con nuestro pool de workers
func InitHttpServerWithWorkerPool(workerPool *worker.WorkerPool, db *gorm.DB, serviceInfo map[string]string) (*gin.Engine, *gin.RouterGroup) {
	// Configurar modo de Gin basado en el entorno
	if gin.Mode() != gin.ReleaseMode {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.New() // Usamos New en lugar de Default para más control sobre los middleware

	// Middleware de recuperación personalizado con logging mejorado
	r.Use(gin.Recovery())

	// Middleware de logging estándar de Gin
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	}))

	// Primero configuramos el middleware de workers para las rutas que lo necesitan
	// Solo rutas específicas usarán el pool de workers para procesar solicitudes en paralelo
	apiProcessingGroup := r.Group("/")
	{
		// Usar el middleware de worker compartido, excluyendo rutas de salud/métricas
		apiProcessingGroup.Use(worker.WorkerPoolMiddleware(
			workerPool, // pool a usar
			5000,       // timeout middleware en ms
			30000,      // timeout procesamiento en ms
			[]string{"/health", "/metrics", "/swagger"}, // rutas a excluir
		))

		// Configurar las rutas que necesitan procesamiento asíncrono
		// Las rutas específicas se configurarán automáticamente al inicializar routers
	}

	// Configurar middleware de trazabilidad para todas las rutas
	//r.Use(worker.TrazabilidadMiddleware(workerPool))

	// Agrupar rutas normales bajo el prefijo de servicio
	api := r.Group(serviceInfo["SERVICE_NAME"])

	// Documentación Swagger
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Configurar todas las rutas normales
	//router.InitializeAllRouters(api, db)

	// Endpoints de estado para monitoreo
	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// Endpoint de métricas usando el handler proporcionado por la biblioteca compartida
	api.GET("/metrics", worker.MetricsEndpoint(workerPool))

	// Iniciar el servidor HTTP
	logger.GetLogger().Info("Iniciando servidor HTTP",
		zap.String("puerto", serviceInfo["SERVICE_PORT"]),
		zap.Int("workers_disponibles", len(workerPool.Workers)))

	return r, api
}
