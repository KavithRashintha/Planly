package config

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

// Load parses environment variables into the given struct based on envconfig tags.
func Load(prefix string, target interface{}) error {
	if err := envconfig.Process(prefix, target); err != nil {
		return fmt.Errorf("failed to load configuration with prefix '%s': %w", prefix, err)
	}
	return nil
}
