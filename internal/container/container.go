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
	configDir   string
	debug       bool
}

type ContainerOptFunc func(*ContainerOpt)

func WithRunnerImage(image string) ContainerOptFunc {
	return func(o *ContainerOpt) {
		o.runnerImage = image
	}
}

func WithConfigDir(dir string) ContainerOptFunc {
	return func(o *ContainerOpt) {
		o.configDir = dir
	}
}

func WithDebug(debug bool) ContainerOptFunc {
	return func(o *ContainerOpt) {
		o.debug = debug
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
func (c *Container) Install(ctx context.Context, ext pgxman.InstallExtension) (*ContainerInfo, error) {
	if err := c.checkDocker(ctx); err != nil {
		return nil, err
	}

	pack := pgxman.Pack{
		APIVersion: pgxman.DefaultPackAPIVersion,
		Extensions: []pgxman.PackExtension{
			{
				Name:      ext.Name,
				Path:      ext.Path,
				Version:   ext.Version,
				Options:   ext.Options,
				Overwrite: true, // always overwrite conflicting pacakge in the container
			},
		},
		Postgres: pgxman.Postgres{
			Version: ext.PGVersion,
			Port:    fmt.Sprintf("%s432", ext.PGVersion),
			// TODO: randomize password
			Username: "pgxman",
			Password: "pgxman",
			DBName:   "pgxman",
		},
	}

	runnerDir := filepath.Join(c.Config.configDir, "runner", string(pack.Postgres.Version))
	if err := os.MkdirAll(runnerDir, 0755); err != nil {
		return nil, err
	}

	runnerImage := c.Config.runnerImage
	if runnerImage == "" {
		runnerImage = fmt.Sprintf("%s/%s:%s", defaultRunnerImageBase, pack.Postgres.Version, pgxman.ImageTag())
	}

	info := ContainerInfo{
		RunnerImage:   runnerImage,
		RunnerDir:     runnerDir,
		ContainerName: fmt.Sprintf("pgxman_runner_%s", pack.Postgres.Version),
		Postgres:      pack.Postgres,
	}
	if c.Config.debug {
		info.PackInstallArgs = "--debug"
	}

	c.Logger.Debug("Exporting template files", "dir", runnerDir, "image", runnerImage, "pg_version", pack.Postgres.Version)
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
	if err := copyLocalFiles(&pack, localFilesDir); err != nil {
		return nil, err
	}

	var (
		packFile    = filepath.Join(runnerDir, "pgxman.yaml")
		tmpPackFile = filepath.Join(runnerDir, "pgxman.yaml.tmp")
	)
	if err := mergePackFile(&pack, packFile, tmpPackFile); err != nil {
		return nil, err
	}
	defer os.Remove(tmpPackFile)

	w := c.Logger.Writer(slog.LevelDebug)
	dockerCompose := exec.CommandContext(
		ctx,
		"docker",
		"compose",
		"up",
		"--build",
		"--wait",
		"--wait-timeout", "60",
		"--remove-orphans",
		"--detach",
	)
	dockerCompose.Dir = runnerDir
	dockerCompose.Stdout = w
	dockerCompose.Stderr = w

	c.Logger.Debug("Starting runner container", "command", dockerCompose.String(), "dir", runnerDir)
	if err := dockerCompose.Run(); err != nil {
		dockerComposeLogs := exec.CommandContext(
			ctx,
			"docker",
			"compose",
			"logs",
			info.ContainerName,
		)
		dockerComposeLogs.Dir = runnerDir
		dockerComposeLogs.Stdout = w
		dockerComposeLogs.Stderr = w

		c.Logger.Debug("Showing runner container logs", "command", dockerComposeLogs.String(), "dir", runnerDir)
		if e := dockerComposeLogs.Run(); e != nil {
			err = errors.Join(err, e)
		}

		return nil, err
	}

	if err := cp.Copy(tmpPackFile, packFile); err != nil {
		return nil, err
	}

	return &info, nil
}

func (c *Container) Teardown(ctx context.Context, pgVer pgxman.PGVersion) error {
	if err := c.checkDocker(ctx); err != nil {
		return err
	}

	runnerDir := filepath.Join(c.Config.configDir, "runner", string(pgVer))
	if _, err := os.Stat(runnerDir); err != nil {
		return fmt.Errorf("runner configuration does not exist: %w", err)
	}

	w := c.Logger.Writer(slog.LevelDebug)
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
	dockerCompose.Stdout = w
	dockerCompose.Stderr = w
	if err := dockerCompose.Run(); err != nil {
		return err
	}

	return os.RemoveAll(runnerDir)
}

func (c *Container) checkDocker(ctx context.Context) error {
	return docker.CheckInstall(ctx)
}

func copyLocalFiles(f *pgxman.Pack, dstDir string) error {
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	var exts []pgxman.PackExtension
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

func mergePackFile(new *pgxman.Pack, packFile, tmpPackFile string) error {
	b, err := os.ReadFile(packFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return writePackFile(new, tmpPackFile)
		} else {
			return err
		}
	}

	var existing pgxman.Pack
	if err := yaml.Unmarshal(b, &existing); err != nil {
		return err
	}

	// new extensions overwrite existing extensions
	extsMap := make(map[string]pgxman.PackExtension)
	for _, ext := range append(existing.Extensions, new.Extensions...) {
		extsMap[ext.Name] = ext
	}
	var result []pgxman.PackExtension
	for _, ext := range extsMap {
		result = append(result, ext)
	}

	// output extensions by name asc
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	new.Extensions = result
	new.Postgres = existing.Postgres // always preserve existing postgres config

	return writePackFile(new, tmpPackFile)
}

func writePackFile(f *pgxman.Pack, dst string) error {
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
	RunnerImage     string
	RunnerDir       string
	ContainerName   string
	Postgres        pgxman.Postgres
	PackInstallArgs string
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
