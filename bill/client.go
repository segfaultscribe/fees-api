// Package bill is the top level service and API endpoint layer
package bill

import (
	"go.temporal.io/sdk/client"
)

// Service represents the running encore service.
//
//encore:service
type Service struct {
	temporalClient client.Client
}

// initService is called by Encore on service startup to establish
// the connection to the Temporal server.
func initService() (*Service, error) {
	c, err := client.Dial(client.Options{
		HostPort:  "localhost:7233",
		Namespace: "default",
	})
	if err != nil {
		return nil, err
	}
	return &Service{temporalClient: c}, nil
}
