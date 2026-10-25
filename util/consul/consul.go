package consul

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/medfriend/shared-commons-go/util/logger"
	"go.uber.org/zap"
	"net/http"
	"os"
)

// newClient crea el cliente; addressDeploy vacio usa la configuracion por defecto.
func newClient(addressDeploy string) *api.Client {
	config := api.DefaultConfig()

	if addressDeploy != "" {
		config.Address = addressDeploy
	} else {
		// Usar la configuración por defecto de Consul
		logger.GetLogger().Info("Usando configuración por defecto de Consul", zap.String("address", config.Address))
	}

	client, err := api.NewClient(config)
	if err != nil {
		logger.GetLogger().Fatal("Error creating consul client", zap.Error(err))
	}
	return client
}

// Connect crea el cliente, lee la ficha de la clave key del KV (la que publica
// getway-go) y registra el servicio con ella. Devuelve la ficha para que el servicio
// tenga su SERVICE_ID y pueda llamar a Deregister al apagar. Si la ficha trae
// SERVICE_HEALTH_PATH el registro lleva chequeo HTTP; si no, queda como antes.
func Connect(addressDeploy, key string) (*api.Client, Ficha, error) {
	client := newClient(addressDeploy)

	ficha, err := GetFicha(client, key)
	if err != nil {
		return nil, nil, err
	}

	id, err := Register(client, ficha)
	if err != nil {
		return nil, nil, err
	}

	logger.GetLogger().Info("Service registered successfully",
		zap.String("service_id", id),
		zap.Bool("health_check", ficha[FichaHealthPath] != ""))

	return client, ficha, nil
}

// ConnectToConsulKey es Connect para quien no necesita la ficha: termina el proceso si
// no puede registrarse, como siempre.
func ConnectToConsulKey(addressDeploy, key string) *api.Client {
	client, _, err := Connect(addressDeploy, key)
	if err != nil {
		logger.GetLogger().Fatal("Error registering service", zap.String("key", key), zap.Error(err))
	}
	return client
}

// ConnectToConsul registra el servicio con las variables de entorno SERVICE_ID,
// SERVICE_NAME, SERVICE_ADDRESS, SERVICE_PORT y, opcional, SERVICE_HEALTH_PATH.
func ConnectToConsul(addressDeploy string) *api.Client {
	client := newClient(addressDeploy)

	ficha := Ficha{
		FichaID:         os.Getenv("SERVICE_ID"),
		FichaName:       os.Getenv("SERVICE_NAME"),
		FichaAddress:    os.Getenv("SERVICE_ADDRESS"),
		FichaPort:       os.Getenv("SERVICE_PORT"),
		FichaHealthPath: os.Getenv("SERVICE_HEALTH_PATH"),
	}

	if ficha[FichaID] == "" || ficha[FichaName] == "" || ficha[FichaAddress] == "" || ficha[FichaPort] == "" {
		logger.GetLogger().Fatal("Missing required environment variables")
	}

	if _, err := Register(client, ficha); err != nil {
		logger.GetLogger().Fatal("Error registering service", zap.Error(err))
	}

	logger.GetLogger().Info("Service registered successfully")

	return client
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Service is healthy"))
}

// GetServiceAddressAndPort devuelve una instancia sana del servicio, sin cache. Para
// resolver en cada peticion conviene NewHealthResolver.
func GetServiceAddressAndPort(client *api.Client, serviceName string) (string, int, error) {
	instances, err := HealthyInstances(client)(serviceName)
	if err != nil {
		return "", 0, err
	}
	if len(instances) == 0 {
		return "", 0, fmt.Errorf("no instances found for service: %s", serviceName)
	}

	logger.GetLogger().Debug("Service name found", zap.String("service_name", serviceName))

	return instances[0].Address, instances[0].Port, nil
}

func StoreKeyValue(client *api.Client, key string, value string) error {

	kv := client.KV()

	p := &api.KVPair{
		Key:   key,
		Value: []byte(value),
	}

	_, err := kv.Put(p, nil)
	if err != nil {
		return fmt.Errorf("error storing key-value pair in Consul: %v", err)
	}

	// Solo clave y tamano: las fichas llevan contrasenas (DB, SMTP, JWT) y antes quedaban en el log.
	logger.GetLogger().Info("Stored key-value pair", zap.String("key", key), zap.Int("bytes", len(value)))
	return nil
}

func GetKeyValue(client *api.Client, key string) (string, error) {
	kv := client.KV()

	pair, _, err := kv.Get(key, nil)
	if err != nil {
		return "", fmt.Errorf("error retrieving key-value pair from Consul: %v", err)
	}

	if pair == nil {
		return "", fmt.Errorf("key %s not found in Consul", key)
	}

	value := string(pair.Value)
	return value, nil
}
