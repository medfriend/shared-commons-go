package rabbitmq

import (
	"github.com/medfriend/shared-commons-go/util/logger"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
	"sync"
)

// NotifyExchange es el exchange por el que viajan las notificaciones dirigidas
// a un usuario.
//
// Existe porque publicar al exchange por defecto con la routing key igual al id
// del usuario deja los mensajes en una cola clásica: si el usuario tiene la app
// abierta en varias ventanas, todas son consumidores en competencia y RabbitMQ
// reparte en round-robin, así que las respuestas caen en la ventana equivocada.
// Publicando aquí, cada conexión obtiene su propia cola exclusiva y recibe una
// copia de todo lo suyo.
const NotifyExchange = "mdf.notify"

type RabbitMQ struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	mu   sync.Mutex // Para manejar acceso concurrente
}

// UserRoutingKey devuelve la routing key de las notificaciones de un usuario.
func UserRoutingKey(userID string) string {
	return "user." + userID
}

// TabRoutingKey devuelve la routing key de una pestaña concreta de un usuario.
// Se usa para respuestas de protocolo (por ejemplo los acks de una carga por
// chunks), que solo le interesan a la ventana que hizo la petición.
func TabRoutingKey(userID string, tabID string) string {
	return "user." + userID + "." + tabID
}

var instancia *RabbitMQ
var once sync.Once

// GetInstance devuelve la instancia única de RabbitMQ
func GetInstance(rabbitConn string) *RabbitMQ {
	once.Do(func() {
		instancia = &RabbitMQ{}
		instancia.connect(rabbitConn) // Establece la conexión inicial
	})
	return instancia
}

// connect establece la conexión y el canal a RabbitMQ
func (r *RabbitMQ) connect(rabbitConn string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var err error
	r.conn, err = amqp.Dial(rabbitConn)
	if err != nil {
		logger.GetLogger().Fatal("Error al conectar a RabbitMQ", zap.Error(err))
	}

	r.ch, err = r.conn.Channel()
	if err != nil {
		logger.GetLogger().Fatal("Error al abrir un canal", zap.Error(err))
	}
}

// ensureConnection verifica y restablece la conexión si está cerrada
func (r *RabbitMQ) ensureConnection(rabbitConn string) {
	if r.conn == nil || r.conn.IsClosed() || r.ch == nil {
		logger.GetLogger().Info("Conexión o canal cerrado, reconectando...")
		r.connect(rabbitConn)
	}
}

// SendMessage envía un mensaje a la cola especificada
func (r *RabbitMQ) SendMessage(queueName string, message string, rabbitConn string) {
	r.ensureConnection(rabbitConn) // Verifica la conexión antes de enviar

	r.mu.Lock()
	defer r.mu.Unlock()

	// Declarar la cola
	_, err := r.ch.QueueDeclare(
		queueName, // Nombre de la cola
		true,      // Duradera
		false,     // Autodelete
		false,     // Exclusiva
		false,     // Sin esperar
		nil,       // Argumentos adicionales
	)
	if err != nil {
		logger.GetLogger().Fatal("Error al declarar la cola", zap.String("queue", queueName), zap.Error(err))
	}

	// Enviar un mensaje
	err = r.ch.Publish(
		"",        // Intercambio
		queueName, // Clave de enrutamiento
		false,     // Requiere confirmación de entrega
		false,     // Exclusivo
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(message),
		})
	if err != nil {
		logger.GetLogger().Fatal("Error al enviar el mensaje", zap.String("queue", queueName), zap.Error(err))
	}

}

// PublishToUser publica una notificación dirigida a todas las ventanas de un
// usuario, a través de NotifyExchange.
//
// Sustituye a SendMessage(userID, ...) para el canal de retorno hacia el
// navegador. SendMessage se mantiene intacto para las colas de trabajo entre
// servicios, que sí quieren reparto entre consumidores.
func (r *RabbitMQ) PublishToUser(userID string, message string, rabbitConn string) {
	if userID == "" {
		logger.GetLogger().Error("PublishToUser sin userID, mensaje descartado")
		return
	}

	r.publishToNotifyExchange(UserRoutingKey(userID), message, rabbitConn)
}

// PublishToUserTab publica una notificación dirigida a una sola pestaña. Si
// tabID viene vacío se degrada a PublishToUser, de modo que un cliente que
// todavía no informa su pestaña sigue recibiendo.
func (r *RabbitMQ) PublishToUserTab(userID string, tabID string, message string, rabbitConn string) {
	if tabID == "" {
		r.PublishToUser(userID, message, rabbitConn)
		return
	}

	if userID == "" {
		logger.GetLogger().Error("PublishToUserTab sin userID, mensaje descartado")
		return
	}

	r.publishToNotifyExchange(TabRoutingKey(userID, tabID), message, rabbitConn)
}

// publishToNotifyExchange declara NotifyExchange (idempotente) y publica.
//
// A diferencia de SendMessage, un fallo aquí no llama a Fatal: perder una
// notificación no puede tumbar el microservicio que la emite.
func (r *RabbitMQ) publishToNotifyExchange(routingKey string, message string, rabbitConn string) {
	r.ensureConnection(rabbitConn)

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.ch == nil {
		logger.GetLogger().Error("Canal de RabbitMQ no disponible, notificación descartada",
			zap.String("routingKey", routingKey))
		return
	}

	err := r.ch.ExchangeDeclare(
		NotifyExchange, // Nombre
		"topic",        // Tipo
		true,           // Duradero
		false,          // Autodelete
		false,          // Interno
		false,          // Sin esperar
		nil,            // Argumentos adicionales
	)
	if err != nil {
		logger.GetLogger().Error("Error al declarar el exchange de notificaciones",
			zap.String("exchange", NotifyExchange), zap.Error(err))
		return
	}

	err = r.ch.Publish(
		NotifyExchange, // Intercambio
		routingKey,     // Clave de enrutamiento
		false,          // Obligatorio
		false,          // Inmediato
		amqp.Publishing{
			ContentType: "application/json",
			Body:        []byte(message),
		})
	if err != nil {
		logger.GetLogger().Error("Error al publicar la notificación",
			zap.String("exchange", NotifyExchange),
			zap.String("routingKey", routingKey),
			zap.Error(err))
		return
	}
}

// Close cierra la conexión y el canal
func (r *RabbitMQ) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.ch != nil {
		r.ch.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}
}
