package pgxm

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/hydradatabase/pgxm/internal/log"
	"github.com/hydradatabase/pgxm/internal/template/debian"
	"github.com/mholt/archiver/v3"

	"golang.org/x/exp/slog"
)

type Extension struct {
	// required
	APIVersion     string       `json:"apiVersion"`
	Name           string       `json:"name"`
	Source         string       `json:"source"`
	Version        string       `json:"version"`
	PGVersions     []string     `json:"pgVersions"`
	InstallCommand string       `json:"installCommand"`
	Maintainers    []Maintainer `json:"maintainers"`

	// optional
	Arch              []Arch     `json:"arch,omitempty"`
	Platform          []Platform `json:"platform,omitempty"`
	Description       string     `json:"description,omitempty"`
	License           string     `json:"license,omitempty"`
	Keywords          []string   `json:"keywords,omitempty"`
	Homepage          string     `json:"homepage,omitempty"`
	BuildDependencies []string   `json:"buildDependencies,omitempty"`
	Dependencies      []string   `json:"dependencies,omitempty"`

	// override
	Deb Deb `json:"deb,omitempty"`
}

type Arch string

const (
	ArchAmd64  Arch = "amd64"
	ArchAarm64 Arch = "arm64"
)

type Platform string

const (
	PlatformLinux Platform = "linux"
)

type Maintainer struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Deb struct {
	BuildDependencies []string `json:"buildDependencies,omitempty"`
	Dependencies      []string `json:"dependencies,omitempty"`
}

var debianFuncMap = template.FuncMap{
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

func NewPackager(workDir string) Packager {
	return &packager{
		workDir: workDir,
		logger:  log.NewTextLogger(),
	}
}

type Packager interface {
	Package(ctx context.Context, ext Extension) error
}

type packager struct {
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
func (p *packager) Package(ctx context.Context, ext Extension) error {
	workDir := filepath.Join(p.workDir, "target")

	extDir := filepath.Join(workDir, "extension")
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

	if err := p.runPackaging(ctx, extDir); err != nil {
		return fmt.Errorf("failed to run packaging: %w", err)
	}

	return nil
}

func (p *packager) downloadSource(ctx context.Context, ext Extension, dstDir string) (string, error) {
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

func (p *packager) unarchiveSource(ctx context.Context, sourceFile, dstDir string) error {
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

func (p *packager) generateDebian(ext Extension, dstDir string) error {
	return fs.WalkDir(debian.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fs.SkipDir
		}

		dst := filepath.Join(dstDir, path)

		if d.IsDir() {
			if err = os.MkdirAll(dst, 0755); err != nil {
				return fmt.Errorf("cannot mkdir %w", err)
			}

			return nil
		}

		if filepath.Base(path) == "embed.go" {
			return nil
		}

		in, err := debian.FS.Open(path)
		if err != nil {
			return fmt.Errorf("cannot open file %w", err)
		}

		b, err := io.ReadAll(in)
		if err != nil {
			return fmt.Errorf("cannot read file %w", err)
		}

		if err := in.Close(); err != nil {
			return fmt.Errorf("cannot close file %w", err)
		}

		t, err := template.New("").Funcs(debianFuncMap).Parse(string(b))
		if err != nil {
			return fmt.Errorf("cannot parse template %w", err)
		}

		out, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("cannot create dst file %w", err)
		}

		if err := t.Execute(out, ext); err != nil {
			out.Close()
			return fmt.Errorf("cannot execute template %w", err)
		}

		if err = out.Close(); err != nil {
			return fmt.Errorf("cannot close dst file %w", err)
		}

		return nil
	})
}

func (p *packager) runPackaging(ctx context.Context, extDir string) error {
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
