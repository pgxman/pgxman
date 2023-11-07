package container

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	cp "github.com/otiai10/copy"
	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/config"
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

var (
	errDockerNotRunning = errors.New("Docker daemon not running")
	errDockerNotFound   = errors.New("Docker client not found")
)

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
	if err := c.checkDocker(); err != nil {
		return nil, err
	}

	// TODO: allow only one pg version for pgxman.yaml
	if len(f.PGVersions) > 1 {
		return nil, fmt.Errorf("multiple PostgreSQL versions are not supported in container")
	}
	pgVer := f.PGVersions[0]

	runnerDir := filepath.Join(config.ConfigDir(), "runner", string(pgVer))
	if err := os.MkdirAll(runnerDir, 0755); err != nil {
		return nil, err
	}

	runnerImage := c.Config.runnerImage
	if runnerImage == "" {
		runnerImage = fmt.Sprintf("%s/%s", defaultRunnerImageBase, pgVer)
	}

	info := ContainerInfo{
		RunnerImage: runnerImage,
		PGVersion:   string(pgVer),
		Port:        fmt.Sprintf("%s432", pgVer),
		RunnerDir:   runnerDir,
		DataDir:     filepath.Join(runnerDir, "data"),
		ServiceName: fmt.Sprintf("pgxman_runner_%s", pgVer),
		PGUser:      "pgxman",
		PGPassword:  "pgxman",
		PGDatabase:  "pgxman",
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
	dockerCompose.Stdout = os.Stdout
	dockerCompose.Stderr = os.Stderr

	return &info, dockerCompose.Run()
}

func (c *Container) Remove(ctx context.Context, pgVer pgxman.PGVersion) error {
	runnerDir := filepath.Join(config.ConfigDir(), "runner", string(pgVer))
	if _, err := os.Stat(runnerDir); err != nil {
		return err
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
	dockerCompose.Stdout = os.Stdout
	dockerCompose.Stderr = os.Stderr
	if err := dockerCompose.Run(); err != nil {
		return err
	}

	return os.RemoveAll(runnerDir)
}

func (c *Container) checkDocker() error {
	cmd := exec.Command("docker", "version", "--format", "json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return errDockerNotFound
		}

		if strings.Contains(string(out), "Cannot connect to the Docker daemon") {
			return errDockerNotRunning
		}

		return fmt.Errorf("docker error: %s %w", out, err)
	}

	outMap := make(map[string]any)
	if err := json.Unmarshal(out, &outMap); err != nil {
		return errDockerNotRunning
	}

	if _, ok := outMap["Server"]; !ok {
		return errDockerNotRunning
	}

	return nil
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

type ContainerInfo struct {
	RunnerImage string
	PGVersion   string
	Port        string
	DataDir     string
	RunnerDir   string
	ServiceName string
	PGUser      string
	PGPassword  string
	PGDatabase  string
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
