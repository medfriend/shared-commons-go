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
func (r *RabbitMQ) SendMessage(queueName string, message string, rabbitConn string) {
	r.ensureConnection(rabbitConn) // Verifica la conexión antes de enviar

	r.mu.Lock()
	defer r.mu.Unlock()

	// Declarar la cola
	_, err := r.ch.QueueDeclare(
		queueName, // Nombre de la cola
		false,     // Duradera
		false,     // Autodelete
		false,     // Exclusiva
		false,     // Sin esperar
		nil,       // Argumentos adicionales
	)
	if err != nil {
		log.Fatalf("Error al declarar la cola: %s", err)
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
