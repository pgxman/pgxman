package pgxman

import (
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"log/slog"

	cp "github.com/otiai10/copy"
	"github.com/pgxman/pgxman/internal/filepathx"
	"github.com/pgxman/pgxman/internal/log"
	tmpl "github.com/pgxman/pgxman/internal/template"
	"github.com/pgxman/pgxman/internal/template/docker"
	"sigs.k8s.io/yaml"
)

const (
	buildWorkspaceDir = "/root/workspace"
)

type BuilderOptions struct {
	ExtDir    string
	CacheFrom []string
	CacheTo   []string
	Pull      bool
	Parallel  int
	Debug     bool
	NoCache   bool
}

func NewBuilder(opts BuilderOptions) Builder {
	return &dockerBuilder{
		BuilderOptions: opts,
		logger:         log.NewTextLogger(),
	}
}

type Builder interface {
	Build(ctx context.Context, ext Extension) error
}

type dockerBuilder struct {
	logger *log.Logger
	BuilderOptions
}

func (b *dockerBuilder) Build(ctx context.Context, ext Extension) error {
	workDir, err := os.MkdirTemp("", "pgxman-build")
	if err != nil {
		return fmt.Errorf("create work directory: %w", err)
	}
	defer func() {
		if !b.Debug {
			os.Remove(workDir)
		}
	}()

	b.logger.Debug("Building extension", "name", ext.Name, "workdir", workDir)
	if err := b.generateDockerFile(ext, workDir); err != nil {
		return fmt.Errorf("generate Dockerfile: %w", err)
	}

	if err := b.generateExtensionFile(ext, workDir); err != nil {
		return fmt.Errorf("generate extension file: %w", err)
	}

	if err := b.runDockerBuild(ctx, ext, workDir); err != nil {
		return fmt.Errorf("docker build: %w", err)
	}

	if err := b.copyBuild(workDir, b.ExtDir); err != nil {
		return fmt.Errorf("copy build: %w", err)
	}

	if b.Debug {
		if err := b.runDockerDebugBuild(ctx, ext, workDir); err != nil {
			return fmt.Errorf("docker debug build: %w", err)
		}
	}

	return nil
}

func (b *dockerBuilder) generateDockerFile(ext Extension, dstDir string) error {
	logger := b.logger.With("name", ext.Name)
	logger.Debug("Generating Dockerfile")

	return tmpl.ExportFS(docker.FS, dockerFileTemplater{ext}, dstDir)
}

func (b *dockerBuilder) generateExtensionFile(ext Extension, dstDir string) error {
	logger := b.logger.With("name", ext.Name)
	logger.Debug("Generating extension.yaml")

	e, err := yaml.Marshal(ext)
	if err != nil {
		return fmt.Errorf("marshal extension: %w", err)
	}

	return os.WriteFile(filepath.Join(dstDir, "extension.yaml"), e, 0644)
}

func (b *dockerBuilder) runDockerBuild(ctx context.Context, ext Extension, dstDir string) error {
	return b.runDockerCmd(
		ctx,
		dstDir,
		b.dockerBakeArgs(
			ext,
			[]string{"export"},
			[]string{
				"--set",
				fmt.Sprintf("*.platform=%s", dockerPlatforms(ext)),
				"--set",
				fmt.Sprintf("*.args.WORKSPACE_DIR=%s", buildWorkspaceDir),
				"--set",
				"export.output=type=local,dest=./out",
			},
		)...,
	)
}

func (b *dockerBuilder) runDockerDebugBuild(ctx context.Context, ext Extension, dstDir string) error {
	targets := dockerBakeTargets(ext)

	return b.runDockerCmd(
		ctx,
		dstDir,
		b.dockerBakeArgs(
			ext,
			targets,
			[]string{
				"--set",
				"*.target=build",
				"--load",
			},
		)...,
	)
}

func (b *dockerBuilder) runDockerCmd(ctx context.Context, dstDir string, args ...string) error {
	dockerBuild := exec.CommandContext(
		ctx,
		"docker",
		args...,
	)
	dockerBuild.Dir = dstDir
	dockerBuild.Stdout = os.Stdout
	dockerBuild.Stderr = os.Stderr

	b.logger.Debug("Running Docker", slog.String("command", dockerBuild.String()))
	return dockerBuild.Run()
}

func (b *dockerBuilder) copyBuild(workDir, dstDir string) error {
	logger := b.logger.With(slog.String("src", workDir), slog.String("dst", dstDir))
	logger.Debug("Copying build")

	var (
		src = filepath.Join(workDir, "out")
		dst = filepath.Join(dstDir, "out")
	)

	if err := os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	matches, err := filepathx.WalkMatch(src, "*.deb")
	if err != nil {
		return fmt.Errorf("glob built extensions: %w", err)
	}

	for _, match := range matches {
		rel, err := filepath.Rel(src, match)
		if err != nil {
			return fmt.Errorf("relative path: %w", err)
		}

		if err := cp.Copy(
			match,
			filepath.Join(dst, rel),
		); err != nil {
			return fmt.Errorf("copy built extension %s: %w", match, err)
		}
	}

	return nil
}

func (b *dockerBuilder) dockerBakeArgs(ext Extension, targets []string, extraArgs []string) []string {
	var (
		buildTargetArgs []string
		sha             = buildSHA(ext)
	)

	for _, builder := range ext.Builders.Available() {
		bakeTargetName := dockerBakeTargetFromBuilderID(builder.Type)

		buildTargetArgs = append(
			buildTargetArgs,
			"--set",
			fmt.Sprintf("%s.args.BUILD_IMAGE=%s", bakeTargetName, builder.Image),
			"--set",
			fmt.Sprintf("%s.args.BUILD_SHA=%s", bakeTargetName, sha),
			"--set",
			fmt.Sprintf("%s.args.PARALLEL=%d", bakeTargetName, b.Parallel),
			"--set",
			fmt.Sprintf("%s.args.WORKSPACE_DIR=%s", bakeTargetName, buildWorkspaceDir),
		)

		if b.BuilderOptions.Debug {
			buildTargetArgs = append(
				buildTargetArgs,
				"--set",
				fmt.Sprintf("%s.tags=%s", bakeTargetName, dockerDebugImage(builder.Type, ext)),
				"--set",
				fmt.Sprintf("%s.args.PGXMAN_PACK_ARGS=--debug", bakeTargetName),
			)
		}
	}

	args := []string{
		"buildx",
		"bake",
	}
	args = append(args, buildTargetArgs...)
	args = append(args, extraArgs...)

	if b.NoCache {
		args = append(args, "--no-cache")
	} else {
		for _, cacheFrom := range b.CacheFrom {
			args = append(args, "--set", fmt.Sprintf("*.cache-from=%s", cacheFrom))
		}

		for _, cacheTo := range b.CacheTo {
			args = append(args, "--set", fmt.Sprintf("*.cache-to=%s", cacheTo))
		}
	}

	if b.Pull {
		args = append(args, "--pull")
	}

	if b.Debug {
		args = append(args, "--progress=plain")
	}

	args = append(args, targets...)

	return args
}

func dockerPlatforms(ext Extension) string {
	var platform []string
	for _, arch := range ext.Arch {
		platform = append(platform, dockerPlatform(arch))
	}

	return strings.Join(platform, ",")
}

func dockerPlatform(arch Arch) string {
	return fmt.Sprintf("linux/%s", arch)
}

func buildSHA(ext Extension) string {
	extb, err := yaml.Marshal(ext)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%x", sha1.Sum(extb))
}

func dockerDebugImage(p Platform, ext Extension) string {
	imagePath := strings.ReplaceAll(string(p), "_", "/")
	return fmt.Sprintf("pgxman/%s/%s:debug", imagePath, ext.Name)
}

func dockerBakeTargets(ext Extension) []string {
	var result []string
	for _, builder := range ext.Builders.Available() {
		result = append(result, dockerBakeTargetFromBuilderID(builder.Type))
	}

	return result
}

func dockerBakeTargetFromBuilderID(p Platform) string {
	return strings.ReplaceAll(string(p), "_", "-")
}

type dockerFileExtension struct {
	Extension
}

func (e dockerFileExtension) ExportDebianBookwormArtifacts() bool {
	if builders := e.Builders; builders != nil {
		return builders.HasBuilder(PlatformDebianBookworm)
	}

	return false
}

func (e dockerFileExtension) ExportUbuntuJammyArtifacts() bool {
	if builders := e.Builders; builders != nil {
		return builders.HasBuilder(PlatformUbuntuJammy)
	}

	return false
}

func (e dockerFileExtension) ExportUbuntuNobleArtifacts() bool {
	if builders := e.Builders; builders != nil {
		return builders.HasBuilder(PlatformUbuntuNoble)
	}

	return false
}

type dockerFileTemplater struct {
	ext Extension
}

func (d dockerFileTemplater) Render(content []byte, out io.Writer) error {
	t, err := template.New("").Parse(string(content))
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	if err := t.Execute(out, dockerFileExtension{d.ext}); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	return nil
}
