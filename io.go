package pgxman

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"dario.cat/mergo"
	"sigs.k8s.io/yaml"
)

func WriteExtension(path string, ext Extension) error {
	b, err := yaml.Marshal(ext)
	if err != nil {
		return err
	}

	return os.WriteFile(path, b, 0644)
}

func ReadPackFile(path string) (*Pack, error) {
	var (
		pack Pack
		b    []byte
		err  error
	)

	if path == "-" {
		b, err = io.ReadAll(os.Stdin)
	} else {
		b, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(b, &pack); err != nil {
		return nil, err
	}

	return &pack, nil
}

func overrideExtension(ext Extension, overrides map[string]any) (Extension, error) {
	bb, err := yaml.Marshal(overrides)
	if err != nil {
		return ext, err
	}

	var yext Extension
	if err := yaml.Unmarshal(bb, &yext); err != nil {
		return ext, err
	}

	if err := mergo.MergeWithOverwrite(&ext, yext); err != nil {
		return ext, err
	}

	// remove pg overrides that are not defined in the overrides map
	if len(yext.PGVersions) > 0 {
		if extOverrides := ext.Overrides; extOverrides != nil {
			if pgVersOverrides := extOverrides.PGVersions; pgVersOverrides != nil {
				extOverridesResult := make(map[PGVersion]ExtensionOverridable)
				for _, v := range yext.PGVersions {
					o, ok := pgVersOverrides[v]
					if ok {
						extOverridesResult[v] = o
					}
				}

				ext.Overrides.PGVersions = extOverridesResult
			}
		}
	}

	return ext, nil
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

	if err := yaml.Unmarshal(b, &ext); err != nil {
		return ext, err
	}

	if len(overrides) > 0 {
		ext, err = overrideExtension(ext, overrides)
		if err != nil {
			return ext, err
		}
	}

	defExt := NewDefaultExtension()
	// Remove default builders that aren't declared
	// so that mergo only merges those that are declared
	if builders := ext.Builders; builders != nil {
		if !builders.HasBuilder(PlatformDebianBookworm) {
			defExt.Builders.DebianBookworm = nil
		}

		if !builders.HasBuilder(PlatformUbuntuJammy) {
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
