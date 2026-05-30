package httpserver

import (
	"encoding/json"
	"errors"
	"os"
)

type Config struct {
	Name string    `json:"name"`
	Port int       `json:"port"`
	SSL  SSLConfig `json:"ssl"`
}

func (c *Config) SSLEnabled() bool {
	return c.SSL.CertFile != "" && c.SSL.KeyFile != ""
}

func (c Config) Verify() error {
	if c.Port <= 0 {
		return errors.New("port must be greater than 0")
	}

	if c.Name == "" {
		return errors.New("name is required")
	}

	return nil
}

type SSLConfig struct {
	CertFile string `json:"certFile"`
	KeyFile  string `json:"keyFile"`
}

func ConfigFromJson(filePath string) (Config, error) {
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return Config{}, err
	}

	var config Config
	err = json.Unmarshal(bytes, &config)
	if err != nil {
		return Config{}, err
	}

	return config, nil
}
