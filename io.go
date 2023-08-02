package pgxman

import (
	"fmt"
	"io"
	"os"

	"github.com/imdario/mergo"
	"sigs.k8s.io/yaml"
)

func WriteExtension(path string, ext Extension) error {
	b, err := yaml.Marshal(ext)
	if err != nil {
		return err
	}

	return os.WriteFile(path, b, 0644)
}

func ReadPGXManfile(path string) (*PGXManfile, error) {
	var (
		pgxmanf PGXManfile
		b       []byte
		err     error
	)

	if path == "-" {
		b, err = io.ReadAll(os.Stdin)
	} else {
		b, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(b, &pgxmanf); err != nil {
		return nil, err
	}

	return &pgxmanf, nil
}

func ReadExtension(path string, overrides map[string]any) (Extension, error) {
	var ext Extension

	if _, err := os.Stat(path); err != nil {
		return ext, fmt.Errorf("%s not found: %w", path, err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return ext, err
	}

	if len(overrides) > 0 {
		b, err = overrideYamlFields(b, overrides)
		if err != nil {
			return ext, err
		}
	}

	if err := yaml.Unmarshal(b, &ext); err != nil {
		return ext, err
	}

	if err := mergo.Merge(&ext, NewDefaultExtension()); err != nil {
		return ext, err
	}

	if err := ext.Validate(); err != nil {
		return ext, fmt.Errorf("invalid extension: %w", err)
	}

	return ext, nil
}

func overrideYamlFields(b []byte, overrides map[string]any) ([]byte, error) {
	src := make(map[string]any)
	if err := yaml.Unmarshal(b, &src); err != nil {
		return nil, err
	}

	if err := mergo.Merge(&overrides, src); err != nil {
		return nil, err
	}

	return yaml.Marshal(overrides)
}
