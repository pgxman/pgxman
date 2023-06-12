package pgxman

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	tmpl "github.com/hydradatabase/pgxman/internal/template"
	"github.com/hydradatabase/pgxman/internal/template/debian"
	"github.com/hydradatabase/pgxman/internal/template/docker"
	"github.com/mholt/archiver/v3"
	cp "github.com/otiai10/copy"
	"golang.org/x/exp/slog"
	"sigs.k8s.io/yaml"
)

var debianPackageFuncMap = template.FuncMap{
	"maintainers": func(ms []Maintainer) string {
		var maintainers []string
		for _, m := range ms {
			maintainers = append(maintainers, fmt.Sprintf("%s <%s>", m.Name, m.Email))
		}

		return strings.Join(maintainers, ", ")
	},
	"buildDeps": func(ext Extension) string {
		required := []string{
			"debhelper (>= 9)",
			"postgresql-server-dev-all (>= 158~)",
		}

		bdeps := ext.BuildDependencies
		if deps := ext.Deb.BuildDependencies; len(deps) != 0 {
			bdeps = deps
		}

		return strings.Join(append(required, bdeps...), ", ")
	},
	"deps": func(ext Extension) string {
		required := []string{
			"${shlibs:Depends}",
			"${misc:Depends}",
		}

		deps := ext.Dependencies
		if d := ext.Deb.Dependencies; len(d) != 0 {
			deps = d
		}

		return strings.Join(append(required, deps...), ", ")
	},
	"timenow": func() string {
		return time.Now().Format(time.RFC1123Z)
	},
}

type debianPackager struct {
	workDir string
	logger  *slog.Logger
}

// Package generates the following folder structure:
//   - workspace
//     -- pgvector.orig.tar.gz
//     -- extension.yaml
//     -- extension
//     --- Makefile
//     --- src
//     --- debian
//     --- buildkit
func (p *debianPackager) Package(ctx context.Context, ext Extension) error {
	workDir := filepath.Join(p.workDir, "target")

	extDir := filepath.Join(workDir, "debian_build")
	if err := os.MkdirAll(extDir, 0755); err != nil {
		return err
	}

	sourceFile, err := p.downloadSource(ctx, ext, workDir)
	if err != nil {
		return fmt.Errorf("failed to download source: %w", err)
	}

	if err := p.unarchiveSource(ctx, sourceFile, extDir); err != nil {
		return fmt.Errorf("failed to unarchive source: %w", err)
	}

	if err := p.generateDebian(ext, extDir); err != nil {
		return fmt.Errorf("failed to generate debian package: %w", err)
	}

	if err := p.buildDebian(ctx, ext, extDir); err != nil {
		return fmt.Errorf("failed to run packaging: %w", err)
	}

	return nil
}

func (p *debianPackager) downloadSource(ctx context.Context, ext Extension, dstDir string) (string, error) {
	logger := p.logger.With(slog.String("source", ext.Source))
	logger.Info("Downloading source")

	resp, err := http.Get(ext.Source)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	sourceFile := filepath.Join(dstDir, fmt.Sprintf("%s_%s.orig.tar.gz", ext.Name, ext.Version))
	f, err := os.Create(sourceFile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", err
	}

	return sourceFile, nil
}

func (p *debianPackager) unarchiveSource(ctx context.Context, sourceFile, dstDir string) error {
	logger := p.logger.With(slog.String("file", sourceFile))
	logger.Info("Unarchiving source")

	sourceDir := filepath.Join(dstDir, "src")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		return err
	}

	ar, err := archiver.ByExtension(sourceFile)
	if err != nil {
		return err
	}

	c, ok := ar.(archiver.Unarchiver)
	if !ok {
		return fmt.Errorf("format specified by source filename is not an archive format: %s", sourceFile)
	}

	// TODO: support more archive types
	targz, ok := ar.(*archiver.TarGz)
	if ok {
		targz.StripComponents = 1
	}

	return c.Unarchive(sourceFile, sourceDir)
}

func (p *debianPackager) generateDebian(ext Extension, dstDir string) error {
	logger := p.logger.With(slog.String("name", ext.Name), slog.String("version", ext.Version))
	logger.Info("Generating debian package")

	return tmpl.Export(debian.FS, debianPackageTemplate{ext}, dstDir)
}

func (p *debianPackager) buildDebian(ctx context.Context, ext Extension, extDir string) error {
	logger := p.logger.With(slog.String("name", ext.Name), slog.String("version", ext.Version))
	logger.Info("Building debian package")

	buildext := exec.CommandContext(ctx, "pg_buildext", "updatecontrol")
	buildext.Dir = extDir
	buildext.Stdout = os.Stdout
	buildext.Stderr = os.Stderr

	if err := buildext.Run(); err != nil {
		return fmt.Errorf("failed to run pg_buildext updatecontrol: %w", err)
	}

	debuild := exec.CommandContext(
		ctx,
		"debuild",
		"--prepend-path", "/usr/local/bin",
		"--preserve-envvar", "CONF_EXTRA_VERSION",
		"--preserve-envvar", "UNENCRYPTED_PACKAGE",
		"--preserve-envvar", "PACKAGE_ENCRYPTION_KEY",
		"--preserve-envvar", "MSRUSTUP_PAT",
		"--preserve-envvar", "MSCODEHUB_USERNAME",
		"--preserve-envvar", "MSCODEHUB_PASSWORD",
		"-uc", "-us", "-B", "--lintian-opts", "--profile", "debian", "--allow-root",
	)
	debuild.Dir = extDir
	debuild.Stdout = os.Stdout
	debuild.Stderr = os.Stderr

	if err := debuild.Run(); err != nil {
		return fmt.Errorf("failed to run debuild: %w", err)
	}

	return nil
}

type debianPackageTemplate struct {
	ext Extension
}

func (d debianPackageTemplate) Apply(content []byte, out io.Writer) error {
	t, err := template.New("").Funcs(debianPackageFuncMap).Parse(string(content))
	if err != nil {
		return fmt.Errorf("cannot parse template %w", err)
	}

	if err := t.Execute(out, d.ext); err != nil {
		return fmt.Errorf("cannot execute template %w", err)
	}

	return nil
}

type debianBuilder struct {
	extDir string
	logger *slog.Logger
}

func (b *debianBuilder) Build(ctx context.Context, ext Extension) error {
	workDir, err := os.MkdirTemp("", "pgxm-build")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.Remove(workDir)

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
		fmt.Sprintf("BUILD_SHA=%s", ext.ConfigSHA),
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

	if err := cp.Copy(
		src,
		dst,
		cp.Options{
			Skip: func(fi os.FileInfo, src, dest string) (bool, error) {
				if fi.IsDir() {
					return false, nil
				}

				return filepath.Ext(fi.Name()) != ".deb", nil
			},
		},
	); err != nil {
		return fmt.Errorf("failed to copy built extensions: %w", err)
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
