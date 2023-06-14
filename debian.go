package pgxman

import (
	"context"
	"crypto/sha1"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tmpl "github.com/hydradatabase/pgxman/internal/template"
	"github.com/hydradatabase/pgxman/internal/template/docker"
	cp "github.com/otiai10/copy"
	"golang.org/x/exp/slog"
	"sigs.k8s.io/yaml"
)

type debianBuilder struct {
	extDir string
	logger *slog.Logger
	debug  bool
}

func (b *debianBuilder) Build(ctx context.Context, ext Extension) error {
	workDir, err := os.MkdirTemp("", "pgxman-build")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer func() {
		if !b.debug {
			os.Remove(workDir)
		}
	}()

	if err := b.generateDockerFile(ext, workDir); err != nil {
		return fmt.Errorf("failed to generate debian package: %w", err)
	}

	if err := b.generateExtensionFile(ext, workDir); err != nil {
		return fmt.Errorf("failed to generate debian package: %w", err)
	}

	if err := b.runDockerBuild(ctx, ext, workDir); err != nil {
		return fmt.Errorf("failed to run docker build: %w", err)
	}

	if err := b.copyBuild(ctx, workDir, b.extDir); err != nil {
		return fmt.Errorf("failed to run docker build: %w", err)
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
		return fmt.Errorf("failed to marshal extension: %w", err)
	}

	return os.WriteFile(filepath.Join(dstDir, "extension.yaml"), e, 0644)
}

func (b *debianBuilder) runDockerBuild(ctx context.Context, ext Extension, dstDir string) error {
	dockerBuild := exec.CommandContext(
		ctx,
		"docker",
		"buildx",
		"build",
		"--build-arg",
		fmt.Sprintf("BUILD_IMAGE=%s", ext.BuildImage),
		"--build-arg",
		fmt.Sprintf("BUILD_SHA=%s", buildSHA(ext)),
		"--platform",
		dockerPlatform(ext),
		"--output",
		"out",
		".",
	)
	dockerBuild.Dir = dstDir
	dockerBuild.Stdout = os.Stdout
	dockerBuild.Stderr = os.Stderr

	logger := b.logger.With(slog.String("name", ext.Name), slog.String("version", ext.Version), slog.String("command", dockerBuild.String()))
	logger.Debug("Running Docker build")
	if err := dockerBuild.Run(); err != nil {
		return fmt.Errorf("failed to run docker build: %w", err)
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
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	matches, err := filepathWalkMatch(src, "*.deb")
	if err != nil {
		return fmt.Errorf("failed to glob built extensions: %w", err)
	}

	for _, match := range matches {
		if err := cp.Copy(
			match,
			filepath.Join(dst, filepath.Base(match)),
		); err != nil {
			return fmt.Errorf("failed to copy built extensions: %w", err)
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
