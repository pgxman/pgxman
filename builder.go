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

	cp "github.com/otiai10/copy"
	"github.com/pgxman/pgxman/internal/filepathx"
	"github.com/pgxman/pgxman/internal/log"
	tmpl "github.com/pgxman/pgxman/internal/template"
	"github.com/pgxman/pgxman/internal/template/docker"
	"golang.org/x/exp/slog"
	"sigs.k8s.io/yaml"
)

type BuilderOptions struct {
	ExtDir    string
	Debug     bool
	NoCache   bool
	CacheFrom []string
	CacheTo   []string
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
	BuilderOptions
	logger *log.Logger
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

	b.logger.Debug("Building extension", "ext", ext, "workdir", workDir)

	if err := b.generateDockerFile(ext, workDir); err != nil {
		return fmt.Errorf("generate Dockerfile: %w", err)
	}

	if err := b.generateExtensionFile(ext, workDir); err != nil {
		return fmt.Errorf("generate extension file: %w", err)
	}

	if err := b.runDockerBuild(ctx, ext, workDir); err != nil {
		return fmt.Errorf("docker build: %w", err)
	}

	if err := b.copyBuild(ctx, workDir, b.ExtDir); err != nil {
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
	logger := b.logger.With(slog.String("name", ext.Name), slog.String("version", ext.Version))
	logger.Debug("Generating Dockerfile")

	return tmpl.Export(docker.FS, dockerFileTemplater{ext}, dstDir)
}

func (b *dockerBuilder) generateExtensionFile(ext Extension, dstDir string) error {
	logger := b.logger.With(slog.String("name", ext.Name), slog.String("version", ext.Version))
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

func (b *dockerBuilder) copyBuild(ctx context.Context, workDir, dstDir string) error {
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

	for _, builder := range ext.Builders.Items() {
		bakeTargetName := dockerBakeTargetFromBuilderID(builder.Type)

		buildTargetArgs = append(
			buildTargetArgs,
			"--set",
			fmt.Sprintf("%s.args.BUILD_IMAGE=%s", bakeTargetName, builder.Image),
			"--set",
			fmt.Sprintf("%s.args.BUILD_SHA=%s", bakeTargetName, sha),
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

func dockerDebugImage(bt ExtensionBuilderType, ext Extension) string {
	var (
		imagePath = strings.ReplaceAll(string(bt), ":", "/")
		///  Docker tags must match the regex [a-zA-Z0-9_.-], which allows alphanumeric characters, dots, underscores, and hyphens.
		tag = strings.ReplaceAll(ext.Version, "+", "-")
	)

	return fmt.Sprintf("pgxman/%s/%s-debug:%s", imagePath, ext.Name, tag)
}

func dockerBakeTargets(ext Extension) []string {
	var result []string
	for _, builder := range ext.Builders.Items() {
		result = append(result, dockerBakeTargetFromBuilderID(builder.Type))
	}

	return result
}

func dockerBakeTargetFromBuilderID(bt ExtensionBuilderType) string {
	return strings.ReplaceAll(string(bt), ":", "-")
}

type dockerFileExtension struct {
	Extension
}

func (e dockerFileExtension) ExportDebianBookwormArtifacts() bool {
	if builders := e.Builders; builders != nil {
		return builders.HasBuilder(ExtensionBuilderDebianBookworm)
	}

	return false
}

func (e dockerFileExtension) ExportUbuntuJammyArtifacts() bool {
	if builders := e.Builders; builders != nil {
		return builders.HasBuilder(ExtensionBuilderUbuntuJammy)
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
