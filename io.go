package pgxman

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"dario.cat/mergo"
	"sigs.k8s.io/yaml"
)

type IO struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func NewStdIO() IO {
	return IO{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

func WriteExtension(path string, ext Extension) error {
	b, err := yaml.Marshal(ext)
	if err != nil {
		return err
	}

	return os.WriteFile(path, b, 0644)
}

func ReadBundleFile(path string) (*Bundle, error) {
	var (
		bundle Bundle
		b      []byte
		err    error
	)

	if path == "-" {
		b, err = io.ReadAll(os.Stdin)
	} else {
		b, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(b, &bundle); err != nil {
		return nil, err
	}

	return &bundle, nil
}

func ReadExtension(path string, overrides map[string]any) (Extension, error) {
	var ext Extension

	path, err := filepath.Abs(path)
	if err != nil {
		return ext, fmt.Errorf("absolute path: %w", err)
	}

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

	defExt := NewDefaultExtension()
	// Remove default builders that aren't declared
	// so that mergo only merges those that are declared
	if builders := ext.Builders; builders != nil {
		if !builders.HasBuilder(ExtensionBuilderDebianBookworm) {
			defExt.Builders.DebianBookworm = nil
		}

		if !builders.HasBuilder(ExtensionBuilderUbuntuJammy) {
			defExt.Builders.UbuntuJammy = nil
		}
	}

	if err := mergo.Merge(
		&ext,
		defExt,
	); err != nil {
		return ext, err
	}

	ext.Path = path

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
