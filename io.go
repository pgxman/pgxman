package pgxman

import (
	"crypto/sha1"
	"fmt"
	"os"

	"github.com/imdario/mergo"
	"sigs.k8s.io/yaml"
)

func WriteExtensionFile(path string, ext Extension) error {
	b, err := yaml.Marshal(ext)
	if err != nil {
		return err
	}

	return os.WriteFile(path, b, 0644)
}

func ReadExtensionFile(path string, overrides map[string]any) (Extension, error) {
	var ext Extension

	if _, err := os.Stat(path); err != nil {
		return ext, fmt.Errorf("extension.yaml not found in current directory")
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return ext, err
	}

	b, err = overrideYamlFields(b, overrides)
	if err != nil {
		return ext, err
	}

	if err := yaml.Unmarshal(b, &ext); err != nil {
		return ext, err
	}

	ext.WithDefaults()
	ext.ConfigSHA = fmt.Sprintf("%x", sha1.Sum(b))

	if err := ext.Validate(); err != nil {
		return ext, fmt.Errorf("invalid extension: %w", err)
	}

	return ext, nil
}

func overrideYamlFields(b []byte, overrides map[string]any) ([]byte, error) {
	if len(overrides) == 0 {
		return b, nil
	}

	src := make(map[string]any)
	if err := yaml.Unmarshal(b, &src); err != nil {
		return nil, err
	}

	if err := mergo.Merge(&overrides, src); err != nil {
		return nil, err
	}

	return yaml.Marshal(overrides)
}
