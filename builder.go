package pgxman

import (
	"context"
	"crypto/sha1"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
	logger := b.logger.With(slog.String("name", ext.Name), slog.String("version", ext.Version))

	commonArgs := []string{
		"buildx",
		"build",
		"--build-arg",
		fmt.Sprintf("BUILD_IMAGE=%s", ext.BuildImage),
		"--build-arg",
		fmt.Sprintf("BUILD_SHA=%s", buildSHA(ext)),
		"--platform",
		dockerPlatform(ext),
	}
	buildArgs := append(
		commonArgs,
		"--output",
		"out",
		".",
	)

	toBuild := [][]string{buildArgs}
	if b.debug {
		tag := fmt.Sprintf("pgxman/%s-debug:%s", ext.Name, ext.Version)
		debugBuildArgs := append(
			commonArgs,
			"--tag",
			tag,
			"--target",
			"build",
			"--load",
			".",
		)
		toBuild = append(toBuild, debugBuildArgs)
		logger.Debug("Docker debug build enabled", slog.String("tag", tag))
	}

	for _, args := range toBuild {
		dockerBuild := exec.CommandContext(
			ctx,
			"docker",
			args...,
		)
		dockerBuild.Dir = dstDir
		dockerBuild.Stdout = os.Stdout
		dockerBuild.Stderr = os.Stderr

		logger.Debug("Docekr build", slog.String("command", dockerBuild.String()))
		if err := dockerBuild.Run(); err != nil {
			return err
		}
	}

	return nil
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

	matches, err := filepathWalkMatch(src, "*.deb")
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

func dockerPlatform(ext Extension) string {
	var platform []string
	for _, arch := range ext.Arch {
		platform = append(platform, fmt.Sprintf("linux/%s", arch))
	}

	return strings.Join(platform, ",")
}

func filepathWalkMatch(root, pattern string) ([]string, error) {
	var matches []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if matched, err := filepath.Match(pattern, filepath.Base(path)); err != nil {
			return err
		} else if matched {
			matches = append(matches, path)
		}

		return nil
	})

	return matches, err
}

func buildSHA(ext Extension) string {
	extb, err := yaml.Marshal(ext)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%x", sha1.Sum(extb))
}
