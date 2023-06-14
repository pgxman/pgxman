package pgxman

import (
	"context"
	"crypto/sha1"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hydradatabase/pgxman/internal/filepathx"
	"github.com/hydradatabase/pgxman/internal/log"
	tmpl "github.com/hydradatabase/pgxman/internal/template"
	"github.com/hydradatabase/pgxman/internal/template/docker"
	cp "github.com/otiai10/copy"
	"golang.org/x/exp/slog"
	"sigs.k8s.io/yaml"
)

func NewBuilder(extDir string, debug bool) Builder {
	return &debianBuilder{
		extDir: extDir,
		logger: log.NewTextLogger(),
		debug:  debug,
	}
}

type Builder interface {
	Build(ctx context.Context, ext Extension) error
}

type debianBuilder struct {
	extDir string
	logger *slog.Logger
	debug  bool
}

func (b *debianBuilder) Build(ctx context.Context, ext Extension) error {
	workDir, err := os.MkdirTemp("", "pgxman-build")
	if err != nil {
		return fmt.Errorf("create work directory: %w", err)
	}
	defer func() {
		if !b.debug {
			os.Remove(workDir)
		}
	}()

	if err := b.generateDockerFile(ext, workDir); err != nil {
		return fmt.Errorf("generate Dockerfile: %w", err)
	}

	if err := b.generateExtensionFile(ext, workDir); err != nil {
		return fmt.Errorf("generate extension file: %w", err)
	}

	if err := b.runDockerBuild(ctx, ext, workDir); err != nil {
		return fmt.Errorf("docker build: %w", err)
	}

	if err := b.copyBuild(ctx, workDir, b.extDir); err != nil {
		return fmt.Errorf("copy build: %w", err)
	}

	if b.debug {
		if err := b.runDockerDebugBuild(ctx, ext, workDir); err != nil {
			return fmt.Errorf("docker debug build: %w", err)
		}
	}

	return nil
}

func (b *debianBuilder) generateDockerFile(ext Extension, dstDir string) error {
	logger := b.logger.With(slog.String("name", ext.Name), slog.String("version", ext.Version))
	logger.Debug("Generating Dockerfile")

	return tmpl.Export(docker.FS, nil, dstDir)
}

func (b *debianBuilder) generateExtensionFile(ext Extension, dstDir string) error {
	logger := b.logger.With(slog.String("name", ext.Name), slog.String("version", ext.Version))
	logger.Debug("Generating extension.yaml")

	e, err := yaml.Marshal(ext)
	if err != nil {
		return fmt.Errorf("marshal extension: %w", err)
	}

	return os.WriteFile(filepath.Join(dstDir, "extension.yaml"), e, 0644)
}

func (b *debianBuilder) runDockerBuild(ctx context.Context, ext Extension, dstDir string) error {
	return b.runDockerCmd(
		ctx,
		dstDir,
		append(
			dockerBuildCommonArgs(ext),
			//"--no-cache",
			"--output",
			"out",
			"--platform",
			dockerPlatforms(ext),
			".",
		)...,
	)
}

func (b *debianBuilder) runDockerDebugBuild(ctx context.Context, ext Extension, dstDir string) error {
	return b.runDockerCmd(
		ctx,
		dstDir,
		append(
			dockerBuildCommonArgs(ext),
			"--tag",
			fmt.Sprintf("pgxman/%s-debug:%s", ext.Name, ext.Version),
			"--target",
			"build",
			"--load",
			".",
		)...,
	)
}

func (b *debianBuilder) runDockerCmd(ctx context.Context, dstDir string, args ...string) error {
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

func (b *debianBuilder) copyBuild(ctx context.Context, workDir, dstDir string) error {
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
		if err := cp.Copy(
			match,
			filepath.Join(dst, filepath.Base(match)),
		); err != nil {
			return fmt.Errorf("copy built extension %s: %w", match, err)
		}
	}

	return nil
}

func dockerBuildCommonArgs(ext Extension) []string {
	return []string{
		"buildx",
		"build",
		"--build-arg",
		fmt.Sprintf("BUILD_IMAGE=%s", ext.BuildImage),
		"--build-arg",
		fmt.Sprintf("BUILD_SHA=%s", buildSHA(ext)),
	}
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
