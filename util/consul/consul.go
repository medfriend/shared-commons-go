package consul

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"log"
	"net/http"
	"os"
	"strconv"
)

func ConnectToConsul() *api.Client {

	client, err := api.NewClient(api.DefaultConfig())

	if err != nil {
		log.Fatalf("Error creating consul client: %v", err)
	}

	serviceID := os.Getenv("SERVICE_ID")
	serviceName := os.Getenv("SERVICE_NAME")
	serviceAddress := os.Getenv("SERVICE_ADDRESS")
	servicePort := os.Getenv("SERVICE_PORT")

	if serviceID == "" || serviceName == "" || serviceAddress == "" || servicePort == "" {
		log.Fatalf("Missing required environment variables")
	}

	port, err := strconv.Atoi(servicePort)
	if err != nil {
		log.Fatalf("Error converting SERVICE_PORT to int: %v", err)
	}

	//TODO implementar el healthcheck
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
		log.Fatalf("Error registering service: %v", err)
	}

	fmt.Println("Service registered successfully")

	return client
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Service is healthy"))
}

func GetServiceAddressAndPort(serviceName string) (string, int, error) {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return "", 0, fmt.Errorf("error creating Consul client: %v", err)
	}

	services, _, err := client.Catalog().Service(serviceName, "", nil)
	if err != nil {
		return "", 0, fmt.Errorf("error retrieving service: %v", err)
	}

	if len(services) == 0 {
		return "", 0, fmt.Errorf("no instances found for service: %s", serviceName)
	}

	service := services[0]
	return service.Address, service.ServicePort, nil
}
