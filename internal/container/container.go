package container

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/config"
	"github.com/pgxman/pgxman/internal/log"
	tmpl "github.com/pgxman/pgxman/internal/template"
	"github.com/pgxman/pgxman/internal/template/runner"
	"sigs.k8s.io/yaml"
)

func NewContainer(cfg ContainerConfig) *Container {
	return &Container{
		Config: cfg,
		Logger: log.NewTextLogger(),
	}
}

// runner
// - 16
//   - Dockerfile
//   - pgxman.yaml
//   - compose.yaml
//
// - 15
//   - Dockerfile
//   - pgxman.yaml
//   - compose.yaml
type Container struct {
	Config ContainerConfig
	Logger *log.Logger
}

type ContainerConfig struct {
	RunnerImage string
}

func (c *Container) Install(ctx context.Context, f *pgxman.PGXManfile) error {
	// TODO: consider allowing only one pg version for pgxman.yaml
	if len(f.PGVersions) > 1 {
		return fmt.Errorf("multiple PostgreSQL versions are not supported in container")
	}
	pgVer := f.PGVersions[0]

	runnerDir := filepath.Join(config.ConfigDir(), "runner", string(pgVer))
	if err := os.MkdirAll(runnerDir, 0755); err != nil {
		return err
	}

	runnerImage := c.Config.RunnerImage
	if runnerImage == "" {
		runnerImage = fmt.Sprintf("ghcr.io/pgxman/runner/postgres/%s:latest", pgVer)
	}

	c.Logger.Debug("Exporting template files", "dir", runnerDir, "image", runnerImage, "pg_version", pgVer)
	if err := tmpl.ExportFS(
		runner.FS,
		runnerTemplater{
			data: runnerData{
				RunnerImage: runnerImage,
				PGVersion:   string(pgVer),
				HostPort:    fmt.Sprintf("%s432", pgVer),
			}},
		runnerDir,
	); err != nil {
		return err
	}

	if err := mergeBundleFile(f, runnerDir); err != nil {
		return err
	}

	dockerCompose := exec.CommandContext(
		ctx,
		"docker",
		"compose",
		"up",
		"--build",
		"--wait",
		"--wait-timeout", "10",
		"--remove-orphans",
		"--detach",
	)
	dockerCompose.Dir = runnerDir
	dockerCompose.Stdout = os.Stdout
	dockerCompose.Stderr = os.Stderr

	return dockerCompose.Run()
}

func mergeBundleFile(new *pgxman.PGXManfile, dstDir string) error {
	bundleFile := filepath.Join(dstDir, "pgxman.yaml")

	b, err := os.ReadFile(bundleFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return writeBundleFile(new, bundleFile)
		} else {
			return err
		}
	}

	var existing pgxman.PGXManfile
	if err := yaml.Unmarshal(b, &existing); err != nil {
		return err
	}

	// new extensions overwrite existing extensions
	extsMap := make(map[string]pgxman.InstallExtension)
	for _, ext := range append(existing.Extensions, new.Extensions...) {
		extsMap[ext.Name] = ext
	}
	var result []pgxman.InstallExtension
	for _, ext := range extsMap {
		result = append(result, ext)
	}

	new.Extensions = result

	return writeBundleFile(new, bundleFile)
}

func writeBundleFile(f *pgxman.PGXManfile, dst string) error {
	bb, err := yaml.Marshal(f)
	if err != nil {
		return err
	}

	if err := os.WriteFile(dst, bb, 0644); err != nil {
		return err
	}

	return nil
}

type runnerData struct {
	RunnerImage string
	PGVersion   string
	HostPort    string
}

type runnerTemplater struct {
	data runnerData
}

func (r runnerTemplater) Render(content []byte, out io.Writer) error {
	t, err := template.New("").Parse(string(content))
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	if err := t.Execute(out, r.data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	return nil
}
