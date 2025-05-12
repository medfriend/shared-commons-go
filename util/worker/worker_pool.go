package worker

import (
	"context"
	"log"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// JobContext contiene la información necesaria para procesar una solicitud HTTP
type JobContext struct {
	Request       *http.Request       // Solicitud HTTP que se debe procesar
	ResponseChan  chan *http.Response // Canal para retornar la respuesta HTTP
	Context       context.Context     // Contexto para manejo de cancelación y timeout
	CancelFunc    context.CancelFunc  // Función para cancelar el contexto (por timeout o shutdown)
	Processing    bool                // Flag para saber si el trabajo ya está siendo procesado
	ProcessingMu  sync.Mutex          // Mutex para proteger el acceso concurrente a 'Processing'
	StartTime     time.Time           // Hora en la que se inició el trabajo
	TimeoutMillis int64               // Tiempo máximo permitido para el trabajo
}

// WorkerPool gestiona un grupo de workers para procesar solicitudes HTTP
type WorkerPool struct {
	JobQueue   chan *JobContext // Cola de trabajos
	Workers    []*Worker        // Lista de workers
	MaxWorkers int              // Número máximo de workers
	Wg         sync.WaitGroup   // Para esperar el cierre limpio de todos los workers
	StopChan   chan struct{}    // Señal para detener todo el pool
}

// Worker representa un worker individual que procesa trabajos
type Worker struct {
	ID        int              // Identificador único del worker
	JobQueue  chan *JobContext // Referencia al canal de trabajos
	StopChan  chan struct{}    // Canal para detener este worker
	Pool      *WorkerPool      // Referencia al pool padre
	Busy      bool             // Indicador si está ocupado
	BusyMu    sync.Mutex       // Mutex para controlar acceso a 'Busy'
	StartTime time.Time        // Hora en la que empezó a procesar un trabajo
}

// JobProcessor define una función que procesa una solicitud HTTP
type JobProcessor func(*http.Request) *http.Response

// DefaultJobProcessor es un procesador básico que siempre retorna 200 OK
var DefaultJobProcessor = func(req *http.Request) *http.Response {
	log.Printf("Procesando solicitud: %s (procesador por defecto)", req.URL.Path)
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
	}
}

// NewWorkerPool crea un nuevo pool de workers
// maxWorkers: número de workers concurrentes (0 = usar número de CPUs)
// queueSize: tamaño máximo de la cola de trabajos
func NewWorkerPool(maxWorkers int, queueSize int) *WorkerPool {
	// Si maxWorkers es 0, usar el número de CPUs
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}
	
	// Asegurar un tamaño mínimo de cola
	if queueSize <= 0 {
		queueSize = 100
	}

	pool := &WorkerPool{
		JobQueue:   make(chan *JobContext, queueSize),
		MaxWorkers: maxWorkers,
		StopChan:   make(chan struct{}),
		Workers:    make([]*Worker, maxWorkers),
	}

	// Inicializar y comenzar workers
	for i := 0; i < maxWorkers; i++ {
		worker := &Worker{
			ID:       i,
			JobQueue: pool.JobQueue,
			StopChan: pool.StopChan,
			Pool:     pool,
			Busy:     false,
		}
		pool.Workers[i] = worker
		pool.Wg.Add(1)
		go worker.Start(&pool.Wg, DefaultJobProcessor)
	}

	// Iniciar monitoreo de workers
	go pool.MonitorWorkers()

	return pool
}

// MonitorWorkers monitorea el estado de los workers periódicamente
func (p *WorkerPool) MonitorWorkers() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			busyCount := 0
			for _, worker := range p.Workers {
				worker.BusyMu.Lock()
				if worker.Busy {
					busyCount++
					duration := time.Since(worker.StartTime)
					if duration > 30*time.Second {
						log.Printf("Worker %d ha estado ocupado durante %v segundos", worker.ID, duration.Seconds())
					}
				}
				worker.BusyMu.Unlock()
			}
			
			// Log de estadísticas del pool (comentado para reducir ruido en logs)
			// log.Printf("Estado del pool: %d/%d workers ocupados, %d trabajos en cola",
			//	busyCount, p.MaxWorkers, len(p.JobQueue))
		case <-p.StopChan:
			return
		}
	}
}

// SubmitJob añade un nuevo trabajo al pool y devuelve un canal para recibir la respuesta
func (p *WorkerPool) SubmitJob(req *http.Request, timeoutMillis int64) chan *http.Response {
	responseChan := make(chan *http.Response, 1)
	
	// Usar el contexto de la solicitud si existe, o crear uno nuevo
	parentCtx := req.Context()
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	
	ctx, cancel := context.WithTimeout(parentCtx, time.Duration(timeoutMillis)*time.Millisecond)

	job := &JobContext{
		Request:       req.WithContext(ctx),
		ResponseChan:  responseChan,
		Context:       ctx,
		CancelFunc:    cancel,
		Processing:    false,
		StartTime:     time.Now(),
		TimeoutMillis: timeoutMillis,
	}

	// Enviar trabajo a la cola
	select {
	case p.JobQueue <- job:
		// Trabajo enviado con éxito
	default:
		// Cola llena, manejar rechazo
		cancel()
		log.Printf("Cola de trabajos llena, rechazando solicitud: %s", req.URL.Path)
		resp := &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Status:     "503 Service Unavailable",
		}
		responseChan <- resp
		close(responseChan)
	}

	return responseChan
}

// SubmitBackgroundJob envía un trabajo al pool para procesamiento en segundo plano sin esperar respuesta
// Ideal para operaciones como logging, trazabilidad, o análisis que no requieren respuesta inmediata
func (p *WorkerPool) SubmitBackgroundJob(req *http.Request, timeoutMillis int64) {
	// Usar el contexto de la solicitud si existe, o crear uno nuevo
	parentCtx := req.Context()
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	
	ctx, cancel := context.WithTimeout(parentCtx, time.Duration(timeoutMillis)*time.Millisecond)

	// Canal descartable para jobs en segundo plano
	dummyResponseChan := make(chan *http.Response, 1)

	job := &JobContext{
		Request:       req.WithContext(ctx),
		ResponseChan:  dummyResponseChan,
		Context:       ctx,
		CancelFunc:    cancel,
		Processing:    false,
		StartTime:     time.Now(),
		TimeoutMillis: timeoutMillis,
	}

	// Inicia una goroutine que intentará enviar el trabajo pero que no bloqueará
	go func() {
		// Intentar enviar el trabajo con un timeout corto
		select {
		case p.JobQueue <- job:
			log.Printf("Trabajo en segundo plano enviado para: %s", req.URL.Path)
		case <-time.After(100 * time.Millisecond): // Timeout corto para no bloquear
			cancel()
			log.Printf("No se pudo encolar el trabajo en segundo plano: %s", req.URL.Path)
			close(dummyResponseChan)
		}
	}()
}

// GetWorkerStats retorna estadísticas sobre el estado actual del pool
func (p *WorkerPool) GetWorkerStats() map[string]interface{} {
	busyWorkers := 0
	for _, worker := range p.Workers {
		worker.BusyMu.Lock()
		if worker.Busy {
			busyWorkers++
		}
		worker.BusyMu.Unlock()
	}
	
	utilization := 0.0
	if p.MaxWorkers > 0 {
		utilization = float64(busyWorkers) / float64(p.MaxWorkers) * 100.0
	}
	
	return map[string]interface{}{
		"workers": map[string]interface{}{
			"total":      len(p.Workers),
			"busy":       busyWorkers,
			"queueSize":  len(p.JobQueue),
			"queueLimit": cap(p.JobQueue),
		},
		"performance": map[string]interface{}{
			"utilization": utilization,
		},
	}
}

// Shutdown cierra correctamente el pool de workers
func (p *WorkerPool) Shutdown() {
	close(p.StopChan)
	p.Wg.Wait()
	log.Println("Worker pool apagado correctamente")
}

// Start comienza el ciclo de procesamiento del worker
func (w *Worker) Start(wg *sync.WaitGroup, processor JobProcessor) {
	defer wg.Done()
	log.Printf("Worker %d iniciado", w.ID)

	for {
		select {
		case job := <-w.JobQueue:
			if job == nil {
				continue
			}

			// Marcar como ocupado
			w.BusyMu.Lock()
			w.Busy = true
			w.StartTime = time.Now()
			w.BusyMu.Unlock()

			// Procesar el trabajo
			w.ProcessJob(job, processor)

			// Marcar como libre
			w.BusyMu.Lock()
			w.Busy = false
			w.BusyMu.Unlock()

		case <-w.StopChan:
			log.Printf("Worker %d detenido", w.ID)
			return
		}
	}
}

// ProcessJob procesa una solicitud HTTP individual
func (w *Worker) ProcessJob(job *JobContext, processor JobProcessor) {
	// Marcar el trabajo como en procesamiento
	job.ProcessingMu.Lock()
	if job.Processing {
		// El trabajo ya está siendo procesado por otro worker
		job.ProcessingMu.Unlock()
		return
	}
	job.Processing = true
	job.ProcessingMu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Worker %d recuperado de pánico: %v", w.ID, r)

			// Cerrar el canal de respuesta si aún está abierto
			select {
			case job.ResponseChan <- &http.Response{
				StatusCode: http.StatusInternalServerError,
				Status:     "500 Internal Server Error",
			}:
			default:
			}
		}

		// Cerrar el canal de respuesta después de usarlo para evitar bloqueos
		close(job.ResponseChan)

		// Asegurarse de que cancelFunc se llame para evitar fugas de recursos
		job.CancelFunc()
	}()

	// Comprobar si el contexto ya ha expirado
	select {
	case <-job.Context.Done():
		log.Printf("Worker %d: trabajo cancelado antes de procesamiento: %v", w.ID, job.Context.Err())
		return
	default:
		// Continuar procesamiento
	}

	startTime := time.Now()
	log.Printf("Worker %d procesando solicitud: %s", w.ID, job.Request.URL.Path)

	// Procesar la solicitud utilizando el procesador proporcionado
	response := processor(job.Request)

	// Medir tiempo de procesamiento
	elapsedMs := time.Since(startTime).Milliseconds()
	log.Printf("Worker %d completó el procesamiento en %d ms para: %s",
		w.ID, elapsedMs, job.Request.URL.Path)

	// Enviar respuesta al canal
	job.ResponseChan <- response
}

// La siguiente función es para mantener compatibilidad con la implementación anterior
// Crea workers pero no aprovecha la funcionalidad completa del pool

// CreateWorkers es la función original que crea un conjunto de goroutines worker
// Se mantiene para compatibilidad con código existente
func CreateWorkers(numWorkers int, stop chan struct{}, taskQueue chan *http.Request) {
	for i := 0; i < numWorkers; i++ {
		go func(id int) {
			for {
				select {
				case req := <-taskQueue:
					if req != nil {
						log.Printf("Worker legacy %d procesando solicitud para: %s", id, req.URL.Path)
					}

				case <-stop:
					log.Printf("Worker legacy %d shutting down", id)
					return
				}
			}
		}(i)
	}
}

// HandleShutdown es la función original para manejar el apagado gracioso
// Se mantiene para compatibilidad con código existente
func HandleShutdown(stop chan struct{}, client interface{}) {
	// Este es un placeholder simplificado para mantener compatibilidad
	// La implementación real sería específica para cada aplicación
	log.Println("Legacy shutdown handler registered")
}