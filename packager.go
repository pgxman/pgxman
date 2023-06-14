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

	"github.com/hydradatabase/pgxman/internal/log"
	tmpl "github.com/hydradatabase/pgxman/internal/template"
	"github.com/hydradatabase/pgxman/internal/template/debian"
	"github.com/mholt/archiver/v3"
	"golang.org/x/exp/slog"
)

func NewPackager(workDir string, debug bool) Packager {
	return &debianPackager{
		workDir: workDir,
		logger:  log.NewTextLogger(),
		debug:   debug,
	}
}

type Packager interface {
	Package(ctx context.Context, ext Extension) error
}

type debianPackager struct {
	workDir string
	logger  *slog.Logger
	debug   bool
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
		return fmt.Errorf("download source %s: %w", ext.Source, err)
	}

	if err := p.unarchiveSource(ctx, sourceFile, extDir); err != nil {
		return fmt.Errorf("unarchive source: %w", err)
	}

	if err := p.generateDebianTemplate(ext, extDir); err != nil {
		return fmt.Errorf("generate debian template: %w", err)
	}

	if err := p.buildDebian(ctx, ext, extDir); err != nil {
		return fmt.Errorf("debian build: %w", err)
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
		return fmt.Errorf("source is not an archive format: %s", sourceFile)
	}

	// TODO: support more archive types
	targz, ok := ar.(*archiver.TarGz)
	if ok {
		targz.StripComponents = 1
	}

	return c.Unarchive(sourceFile, sourceDir)
}

func (p *debianPackager) generateDebianTemplate(ext Extension, dstDir string) error {
	logger := p.logger.With(slog.String("name", ext.Name), slog.String("version", ext.Version))
	logger.Info("Generating debian package")

	return tmpl.Export(debian.FS, debianPackageTemplater{ext}, dstDir)
}

func (p *debianPackager) buildDebian(ctx context.Context, ext Extension, extDir string) error {
	logger := p.logger.With(slog.String("name", ext.Name), slog.String("version", ext.Version))
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
	debuild.Dir = extDir
	debuild.Stdout = os.Stdout
	debuild.Stderr = os.Stderr

	if err := debuild.Run(); err != nil {
		return fmt.Errorf("debuild: %w", err)
	}

	return nil
}

type extensionData struct {
	Extension
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
	ext Extension
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
