package config

import (
	"os"
	"path/filepath"
	"time"

	"sigs.k8s.io/yaml"
)

type Config struct {
	LastUpgradeCheckTime time.Time
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

	return &c, nil
}

func configFile() string {
	return filepath.Join(ConfigDir(), "config.yml")
}

func ConfigDir() string {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		panic(err.Error())
	}

	return filepath.Join(userConfigDir, "pgxman")
}
