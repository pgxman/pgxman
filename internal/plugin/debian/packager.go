package debian

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/mholt/archiver/v3"
	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/log"
	tmpl "github.com/pgxman/pgxman/internal/template"
	"github.com/pgxman/pgxman/internal/template/debian"
	"golang.org/x/exp/slog"
)

type DebianPackager struct {
	Logger *log.Logger
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
func (p *DebianPackager) Package(ctx context.Context, ext pgxman.Extension, opts pgxman.PackagerOptions) error {
	workDir := filepath.Join(opts.WorkDir, "target")

	extDir := filepath.Join(workDir, "debian_build")
	if err := os.MkdirAll(extDir, 0755); err != nil {
		return err
	}

	sourceFile, err := p.downloadSource(ctx, ext, workDir)
	if err != nil {
		return fmt.Errorf("download source %s: %w", ext.Source, err)
	}

	if err := p.unarchiveSource(ctx, sourceFile, extDir); err != nil {
		return fmt.Errorf("unarchive source: %w", err)
	}

	if err := p.installBuildDependencies(ctx, ext); err != nil {
		return fmt.Errorf("install build dependencies: %w", err)
	}

	if err := p.generateDebianTemplate(ext, extDir); err != nil {
		return fmt.Errorf("generate debian template: %w", err)
	}

	if err := p.buildDebian(ctx, ext, extDir); err != nil {
		return fmt.Errorf("debian build: %w", err)
	}

	return nil
}

func (p *DebianPackager) downloadSource(ctx context.Context, ext pgxman.Extension, dstDir string) (string, error) {
	logger := p.Logger.With(slog.String("source", ext.Source))
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

func (p *DebianPackager) unarchiveSource(ctx context.Context, sourceFile, dstDir string) error {
	logger := p.Logger.With(slog.String("file", sourceFile))
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
		return fmt.Errorf("source is not an archive format: %s", sourceFile)
	}

	// TODO: support more archive types
	targz, ok := ar.(*archiver.TarGz)
	if ok {
		targz.StripComponents = 1
	}

	return c.Unarchive(sourceFile, sourceDir)
}

func (p *DebianPackager) generateDebianTemplate(ext pgxman.Extension, dstDir string) error {
	logger := p.Logger.With(slog.String("name", ext.Name), slog.String("version", ext.Version))
	logger.Info("Generating debian package")

	return tmpl.Export(debian.FS, debianPackageTemplater{ext}, dstDir)
}

func (p *DebianPackager) installBuildDependencies(ctx context.Context, ext pgxman.Extension) error {
	logger := p.Logger.With(slog.String("name", ext.Name), slog.String("version", ext.Version), slog.Any("dependencies", ext.BuildDependencies))
	logger.Info("Installing build deps")

	aptUpdate := exec.CommandContext(ctx, "apt", "update")
	aptUpdate.Stdout = os.Stdout
	aptUpdate.Stderr = os.Stderr

	if err := aptUpdate.Run(); err != nil {
		return fmt.Errorf("apt update: %w", err)
	}

	aptInstall := exec.CommandContext(ctx, "apt", "install", "-y", strings.Join(ext.BuildDependencies, " "))
	aptInstall.Stdout = os.Stdout
	aptInstall.Stderr = os.Stderr

	if err := aptInstall.Run(); err != nil {
		return fmt.Errorf("apt install: %w", err)
	}

	return nil
}

func (p *DebianPackager) buildDebian(ctx context.Context, ext pgxman.Extension, extDir string) error {
	logger := p.Logger.With(slog.String("name", ext.Name), slog.String("version", ext.Version))
	logger.Info("Building debian package")

	buildext := exec.CommandContext(ctx, "pg_buildext", "updatecontrol")
	buildext.Dir = extDir
	buildext.Stdout = os.Stdout
	buildext.Stderr = os.Stderr

	if err := buildext.Run(); err != nil {
		return fmt.Errorf("pg_buildext updatecontrol: %w", err)
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
	debuild.Env = append(
		os.Environ(),
		fmt.Sprintf("DEB_BUILD_OPTIONS=noautodbgsym parallel=%d", runtime.NumCPU()),
	)
	debuild.Dir = extDir
	debuild.Stdout = os.Stdout
	debuild.Stderr = os.Stderr

	if err := debuild.Run(); err != nil {
		return fmt.Errorf("debuild: %w", err)
	}

	return nil
}

type extensionData struct {
	pgxman.Extension
}

func (e extensionData) Maintainers() string {
	var maintainers []string
	for _, m := range e.Extension.Maintainers {
		maintainers = append(maintainers, fmt.Sprintf("%s <%s>", m.Name, m.Email))
	}

	return strings.Join(maintainers, ", ")
}

func (e extensionData) Deps() string {
	required := []string{
		"${shlibs:Depends}",
		"${misc:Depends}",
	}

	deps := e.Dependencies
	if deb := e.Deb; deb != nil && len(deb.Dependencies) != 0 {
		deps = deb.Dependencies
	}

	return strings.Join(append(required, deps...), ", ")

}

func (e extensionData) BuildDeps() string {
	required := []string{
		"debhelper (>= 9)",
		"postgresql-server-dev-all (>= 158~)",
	}

	bdeps := e.BuildDependencies
	if deb := e.Deb; deb != nil && len(deb.BuildDependencies) != 0 {
		bdeps = deb.BuildDependencies
	}

	return strings.Join(append(required, bdeps...), ", ")
}

func (e extensionData) TimeNow() string {
	return time.Now().Format(time.RFC1123Z)
}

type debianPackageTemplater struct {
	ext pgxman.Extension
}

func (d debianPackageTemplater) Render(content []byte, out io.Writer) error {
	t, err := template.New("").Parse(string(content))
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	if err := t.Execute(out, extensionData{d.ext}); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	return nil
}
