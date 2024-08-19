package config

import "fmt"

type ConfigFileInvalidError struct {
	Err error
}

type ConfigFileNotFoundException struct {
	Path string
	Err  error
}

func (e *ConfigFileInvalidError) Error() string {
	return fmt.Sprintf("Configuration file is invalid: %s", e.Err)
}

func (e *ConfigFileNotFoundException) Error() string {
	return fmt.Sprintf("Configuration file not found: %s", e.Path)
}
