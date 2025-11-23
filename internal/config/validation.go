package config

import "fmt"

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field string
	Value interface{}
	Err   error
}

func (e *ValidationError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("invalid value for %s: %v (%v)", e.Field, e.Value, e.Err)
	}
	return fmt.Sprintf("invalid value for %s: %v", e.Field, e.Value)
}

// Validate validates the entire configuration
func (c *Config) Validate() error {
	// Validate encoding
	if err := c.Encoding.Validate(); err != nil {
		return err
	}

	// Validate transition
	if err := c.Transition.Validate(); err != nil {
		return err
	}

	// Add more validation as needed
	return nil
}
