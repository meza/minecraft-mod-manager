package config

import "fmt"

type FileInvalidError struct {
	Err error
}

type ConfigFileNotFoundException struct {
	Path string
	Err  error
}

func (fileError *FileInvalidError) Error() string {
	return fmt.Sprintf("Configuration file is invalid: %s", fileError.Err)
}

func (configError *ConfigFileNotFoundException) Error() string {
	return fmt.Sprintf("Configuration file not found: %s", configError.Path)
}
