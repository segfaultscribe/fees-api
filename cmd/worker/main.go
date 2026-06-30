// Command worker runs the Temporal worker process that executes
// the BillWorkflow and its associated update and query handlers.
package main

import (
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"encore.app/bill/workflow"
)

const taskQueue = "fees-api-bill-queue"

func main() {
	hostPort := "localhost:7233"
	namespace := "default"

	c, err := client.Dial(client.Options{
		HostPort:  hostPort,
		Namespace: namespace,
	})
	if err != nil {
		log.Fatalf("failed to connect to Temporal: %v", err)
	}
	defer c.Close()

	w := worker.New(c, taskQueue, worker.Options{})
	w.RegisterWorkflow(workflow.BillWorkflow)

	log.Println("starting bill workflow worker on task queue:", taskQueue)
	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalf("worker stopped with error: %v", err)
	}
}
