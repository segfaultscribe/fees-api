package bill

import "go.temporal.io/sdk/client"

// taskQueue is the Temporal task queue used by the bill worker.
const taskQueue = "fees-api-bill-queue"

// temporalClient is the shared client used by all bill API endpoints
// to communicate with the Temporal server.
var temporalClient client.Client

// initService is called by Encore on service startup to establish
// the connection to the Temporal server.
func initService() error {
	c, err := client.Dial(client.Options{
		HostPort:  cfg.TemporalHostPort(),
		Namespace: cfg.TemporalNamespace(),
	})
	if err != nil {
		return err
	}
	temporalClient = c
	return nil
}
