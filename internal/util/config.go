package util

import (
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const DefaultConfigPath = "config.yaml"

type Config struct {
	Environment string `yaml:"environment"`
	App         struct {
		BaseURL string `yaml:"base_url"`
	} `yaml:"app"`
	Server struct {
		Addr string `yaml:"addr"`
	} `yaml:"server"`
	Auth struct {
		MagicLinkTTL  string `yaml:"magic_link_ttl"`
		SessionTTL    string `yaml:"session_ttl"`
		SessionSecret string `yaml:"session_secret"`
	} `yaml:"auth"`
	Log struct {
		Level string `yaml:"level"`
	} `yaml:"log"`
	Database struct {
		DSN string `yaml:"dsn"`
	} `yaml:"database"`
	Email struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		From     string `yaml:"from"`
	} `yaml:"email"`
}

// expandEnv replaces ${VAR} and ${VAR:-default} in s using environment variables.
func expandEnv(s string) string {
	return os.Expand(s, func(key string) string {
		if idx := strings.Index(key, ":-"); idx != -1 {
			if val := os.Getenv(key[:idx]); val != "" {
				return val
			}
			return key[idx+2:]
		}
		return os.Getenv(key)
	})
}

func LoadConfig(filePath string) (Config, error) {
	var config Config

	file, err := os.Open(filePath)
	if err != nil {
		return config, err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return config, err
	}

	expanded := expandEnv(string(content))

	d := yaml.NewDecoder(strings.NewReader(expanded))
	if err := d.Decode(&config); err != nil {
		return config, err
	}
	return config, nil
}
