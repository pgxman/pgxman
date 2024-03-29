package config

import (
	"os"
	"path/filepath"
	"time"

	"dario.cat/mergo"
	"sigs.k8s.io/yaml"
)

type Config struct {
	OAuth                OAuth     `json:"oauth"`
	LastUpgradeCheckTime time.Time `json:"lastUpgradeCheckTime"`
}

type OAuth struct {
	ClientID string `json:"clientId"`
	Audience string `json:"audience"`
	Endpoint string `json:"endpoint"`
}

func Write(c Config) error {
	b, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	cf := configFile()
	if err := os.MkdirAll(filepath.Dir(cf), 0755); err != nil {
		return err
	}

	return os.WriteFile(cf, b, 0644)
}

func Read() (*Config, error) {
	b, err := os.ReadFile(configFile())
	if err != nil {
		return nil, err
	}

	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}

	if err := mergo.Merge(&c, newDefaultConfig()); err != nil {
		return nil, err
	}

	return &c, nil
}

func ConfigDir() string {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		panic(err.Error())
	}

	return filepath.Join(userConfigDir, "pgxman")
}

func newDefaultConfig() Config {
	return Config{
		OAuth: OAuth{
			ClientID: "Zf43BaHXF0LVcnm9ZKvCwMVqyPkddlp6",
			Endpoint: "https://auth.hydra.so",
			Audience: "https://login.pgxman.com",
		},
	}
}

func configFile() string {
	return filepath.Join(ConfigDir(), "config.yml")
}
