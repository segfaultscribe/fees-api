package bill

import (
	"log"

	"go.temporal.io/sdk/client"
)

// taskQueue is the Temporal task queue used by the bill worker.
const taskQueue = "fees-api-bill-queue"

// Service represents the running encore service.
//
//encore:service
type Service struct {
	temporalClient client.Client
}

// initService is called by Encore on service startup to establish
// the connection to the Temporal server.
func initService() (*Service, error) {
	log.Println("initService: connecting to Temporal")

	c, err := client.Dial(client.Options{
		HostPort:  "localhost:7233",
		Namespace: "default",
	})
	if err != nil {
		return nil, err
	}
	return &Service{temporalClient: c}, nil
}
