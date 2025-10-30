package consul

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/medfriend/shared-commons-go/util/logger"
	"go.uber.org/zap"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func handleString2int(data string) int {
	converted, err := strconv.Atoi(data)

	if err != nil {
		logger.GetLogger().Fatal("Error al convertir string a int", zap.Error(err))
	}

	return converted
}

func ConnectToConsulKey(addressDeploy, key string) *api.Client {

	config := api.DefaultConfig()

	if addressDeploy != "" {
		config.Address = addressDeploy
	} else {
		// Usar la configuración por defecto de Consul
		logger.GetLogger().Info("Usando configuración por defecto de Consul", zap.String("address", config.Address))
	}

	client, err := api.NewClient(config)

	dbString, _ := GetKeyValue(client, key)

	var result map[string]string

	err = json.Unmarshal([]byte(dbString), &result)

	if err != nil {
		logger.GetLogger().Fatal("Error converting JSON string to map", zap.Error(err))
	}

	service := &api.AgentServiceRegistration{
		ID:      result["SERVICE_ID"],
		Name:    result["SERVICE_NAME"],
		Address: result["SERVICE_ADDRESS"],
		Port:    handleString2int(result["SERVICE_PORT"]),
		//Check: &api.AgentServiceCheck{
		//	HTTP:     fmt.Sprintf("http://%s:%d/health", serviceAddress, 8080),
		//	Interval: "10s",
		//	Timeout:  "5s",
		//},
	}

	err = client.Agent().ServiceRegister(service)

	if err != nil {
		logger.GetLogger().Fatal("Error registering service", zap.Error(err))
	}

	logger.GetLogger().Info("Service registered successfully")

	return client
}

func ConnectToConsul(addressDeploy string) *api.Client {

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

	serviceID := os.Getenv("SERVICE_ID")
	serviceName := os.Getenv("SERVICE_NAME")
	serviceAddress := os.Getenv("SERVICE_ADDRESS")
	servicePort := os.Getenv("SERVICE_PORT")

	if serviceID == "" || serviceName == "" || serviceAddress == "" || servicePort == "" {
		logger.GetLogger().Fatal("Missing required environment variables")
	}

	port, err := strconv.Atoi(servicePort)
	if err != nil {
		logger.GetLogger().Fatal("Error converting SERVICE_PORT to int", zap.Error(err))
	}

	service := &api.AgentServiceRegistration{
		ID:      serviceID,
		Name:    os.Getenv("SERVICE_NAME"),
		Address: os.Getenv("SERVICE_ADDRESS"),
		Port:    port,
		//Check: &api.AgentServiceCheck{
		//	HTTP:     fmt.Sprintf("http://%s:%d/health", serviceAddress, 8080),
		//	Interval: "10s",
		//	Timeout:  "5s",
		//},
	}

	err = client.Agent().ServiceRegister(service)

	if err != nil {
		logger.GetLogger().Fatal("Error registering service", zap.Error(err))
	}

	logger.GetLogger().Info("Service registered successfully")

	return client
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Service is healthy"))
}

func GetServiceAddressAndPort(client *api.Client, serviceName string) (string, int, error) {

	services, _, err := client.Catalog().Service(serviceName, "", nil)
	if err != nil {
		return "", 0, fmt.Errorf("error retrieving service: %v", err)
	}

	if len(services) == 0 {
		return "", 0, fmt.Errorf("no instances found for service: %s", serviceName)
	}

	service := services[0]

	logger.GetLogger().Debug("Service name found", zap.String("service_name", service.ServiceName))

	parts := strings.Split(service.ServiceName, "-")
	dbString, _ := GetKeyValue(client, strings.ToUpper(parts[1]))

	var result map[string]string

	err = json.Unmarshal([]byte(dbString), &result)

	return result["SERVICE_ADDRESS"], service.ServicePort, nil
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

	logger.GetLogger().Info("Stored key-value pair", zap.String("key", key), zap.String("value", value))
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
