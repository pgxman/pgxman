package container

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"text/template"

	cp "github.com/otiai10/copy"
	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/config"
	"github.com/pgxman/pgxman/internal/docker"
	"github.com/pgxman/pgxman/internal/log"
	tmpl "github.com/pgxman/pgxman/internal/template"
	"github.com/pgxman/pgxman/internal/template/runner"
	"sigs.k8s.io/yaml"
)

const (
	defaultRunnerImageBase = "ghcr.io/pgxman/runner/postgres"
)

func NewContainer(opts ...ContainerOptFunc) *Container {
	cfg := &ContainerOpt{}
	for _, opt := range opts {
		opt(cfg)
	}

	return &Container{
		Config: cfg,
		Logger: log.NewTextLogger(),
	}
}

type Container struct {
	Config *ContainerOpt
	Logger *log.Logger
}

type ContainerOpt struct {
	runnerImage string
}

type ContainerOptFunc func(*ContainerOpt)

func WithRunnerImage(image string) ContainerOptFunc {
	return func(o *ContainerOpt) {
		o.runnerImage = image
	}
}

// Install installs extensions specified in a pgxman.yaml file into a container.
//
// The folder structure of the configuration files is as follows:
//
// - USER_CONFIG_DIR
// --- pgxman
// ----- runner
// ------- {{ .PG_VERSION }}
// --------- Dockerfile
// --------- pgxman.yaml
// --------- compose.yaml
// --------- files
func (c *Container) Install(ctx context.Context, f *pgxman.PGXManfile) (*ContainerInfo, error) {
	if err := c.checkDocker(ctx); err != nil {
		return nil, err
	}

	pgVer := f.Postgres.Version

	// Set default values
	// TODO: randomize password
	f.Postgres.Username = "pgxman"
	f.Postgres.Password = "pgxman"
	f.Postgres.DBName = "pgxman"
	f.Postgres.Port = fmt.Sprintf("%s432", pgVer)

	runnerDir := filepath.Join(config.ConfigDir(), "runner", string(pgVer))
	if err := os.MkdirAll(runnerDir, 0755); err != nil {
		return nil, err
	}

	runnerImage := c.Config.runnerImage
	if runnerImage == "" {
		runnerImage = fmt.Sprintf("%s/%s:%s", defaultRunnerImageBase, pgVer, pgxman.ImageTag())
	}

	info := ContainerInfo{
		RunnerImage:   runnerImage,
		RunnerDir:     runnerDir,
		ContainerName: fmt.Sprintf("pgxman_runner_%s", pgVer),
		Postgres:      f.Postgres,
	}

	c.Logger.Debug("Exporting template files", "dir", runnerDir, "image", runnerImage, "pg_version", pgVer)
	if err := tmpl.ExportFS(
		runner.FS,
		runnerTemplater{
			info: info,
		},
		runnerDir,
	); err != nil {
		return nil, err
	}

	localFilesDir := filepath.Join(runnerDir, "files")
	c.Logger.Debug("Copying local files", "dir", localFilesDir)
	if err := copyLocalFiles(f, localFilesDir); err != nil {
		return nil, err
	}

	if err := mergeBundleFile(f, runnerDir); err != nil {
		return nil, err
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
	dockerCompose.Stdout = c.Logger.Writer(slog.LevelDebug)
	dockerCompose.Stderr = c.Logger.Writer(slog.LevelDebug)

	return &info, dockerCompose.Run()
}

func (c *Container) Teardown(ctx context.Context, pgVer pgxman.PGVersion) error {
	if err := c.checkDocker(ctx); err != nil {
		return err
	}

	runnerDir := filepath.Join(config.ConfigDir(), "runner", string(pgVer))
	if _, err := os.Stat(runnerDir); err != nil {
		return fmt.Errorf("runner configuration does not exist: %w", err)
	}

	dockerCompose := exec.CommandContext(
		ctx,
		"docker",
		"compose",
		"down",
		"--remove-orphans",
		"--timeout", "10",
		"--volumes",
	)
	dockerCompose.Dir = runnerDir
	dockerCompose.Stdout = c.Logger.Writer(slog.LevelDebug)
	dockerCompose.Stderr = c.Logger.Writer(slog.LevelDebug)
	if err := dockerCompose.Run(); err != nil {
		return err
	}

	return os.RemoveAll(runnerDir)
}

func (c *Container) checkDocker(ctx context.Context) error {
	return docker.CheckInstall(ctx)
}

func copyLocalFiles(f *pgxman.PGXManfile, dstDir string) error {
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	var exts []pgxman.InstallExtension
	for _, ext := range f.Extensions {
		if src := ext.Path; src != "" {
			dst := filepath.Join(dstDir, filepath.Base(src))
			if err := cp.Copy(src, dst); err != nil {
				return err
			}

			// rewrite path in the container
			ext.Path = fmt.Sprintf("/pgxman/files/%s", filepath.Base(src))
		}

		exts = append(exts, ext)
	}

	f.Extensions = exts

	return nil
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

	// output extensions by name asc
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	new.Extensions = result
	new.Postgres = existing.Postgres // always preserve existing postgres config

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

type ContainerInfo struct {
	RunnerImage   string
	RunnerDir     string
	ContainerName string
	Postgres      pgxman.Postgres
}

type runnerTemplater struct {
	info ContainerInfo
}

func (r runnerTemplater) Render(content []byte, out io.Writer) error {
	t, err := template.New("").Parse(string(content))
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	if err := t.Execute(out, r.info); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	return nil
}
