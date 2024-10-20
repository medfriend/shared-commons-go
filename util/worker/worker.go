package worker

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func HandleShutdown(stop chan struct{}, client *api.Client) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	serviceID := os.Getenv("SERVICE_ID")

	sig := <-sigChan
	fmt.Printf("Caught signal %s: shutting down gracefully...\n", sig)

	err := client.Agent().ServiceDeregister(serviceID)

	if err != nil {
		log.Fatalf("Error deregistering service: %v", err)
	}

	fmt.Println("Service deregistered successfully")

	close(stop)
	os.Exit(0)

}

func CreateWorkers(numWorkers int, stop chan struct{}, taskQueue chan *http.Request) {
	for i := 0; i < numWorkers; i++ {
		go func(id int) {
			for {
				select {
				case req := <-taskQueue:
					fmt.Printf("Worker %d processing request for: %s\n", id, req.URL.Path)

				case <-stop:
					fmt.Printf("Worker %d shutting down\n", id)
					return
				}
			}
		}(i)
	}
}
