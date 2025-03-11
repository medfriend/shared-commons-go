package rabbitmq

import (
	"github.com/streadway/amqp"
	"log"
	"sync"
)

type RabbitMQ struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	mu   sync.Mutex // Para manejar acceso concurrente
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
		log.Fatalf("Error al conectar a RabbitMQ: %s", err)
	}

	r.ch, err = r.conn.Channel()
	if err != nil {
		log.Fatalf("Error al abrir un canal: %s", err)
	}
}

// ensureConnection verifica y restablece la conexión si está cerrada
func (r *RabbitMQ) ensureConnection(rabbitConn string) {
	if r.conn == nil || r.conn.IsClosed() || r.ch == nil {
		log.Println("Conexión o canal cerrado, reconectando...")
		r.connect(rabbitConn)
	}
}

// SendMessage envía un mensaje a la cola especificada
func (r *RabbitMQ) SendMessage(queueName string, message string, rabbitConn string, exchangeType string, routingKey string) {
	r.ensureConnection(rabbitConn) // Verifica la conexión antes de enviar

	r.mu.Lock()
	defer r.mu.Unlock()

	// Declarar el exchange basado en el tipo proporcionado
	err := r.ch.ExchangeDeclare(
		queueName,    // Nombre del exchange
		exchangeType, // Tipo de exchange: "direct" o "fanout"
		true,         // Duradero
		false,        // Autodelete
		false,        // Exclusivo
		false,        // Sin esperar
		nil,          // Argumentos adicionales
	)
	if err != nil {
		log.Fatalf("Error al declarar el exchange: %s", err)
	}

	// Si el exchange es de tipo 'fanout', ignora la clave de enrutamiento porque no es necesaria
	if exchangeType == "fanout" {
		routingKey = "" // En fanout, la clave de enrutamiento no se usa
	}

	// Enviar un mensaje
	err = r.ch.Publish(
		queueName,  // Nombre del exchange
		routingKey, // Clave de enrutamiento, se usa con 'direct' exchange
		false,      // Requiere confirmación de entrega
		false,      // Exclusivo
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(message),
		})
	if err != nil {
		log.Fatalf("Error al enviar el mensaje: %s", err)
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
