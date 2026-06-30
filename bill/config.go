package bill

import "encore.dev/config"

// Config holds runtime configuration for the bill service.
type Config struct {
	TemporalHostPort  config.String
	TemporalNamespace config.String
}

var cfg = config.Load[*Config]()
