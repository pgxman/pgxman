package debian

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"

	"log/slog"

	"github.com/mholt/archiver/v3"
	cp "github.com/otiai10/copy"
	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/log"
	tmpl "github.com/pgxman/pgxman/internal/template"
	"github.com/pgxman/pgxman/internal/template/debian"
)

const (
	extensionDepPrefix = "pgxman/"
	sourceTarGzPath    = "/workspace/source.tar.gz"
)

type DebianPackager struct {
	Logger *log.Logger
}

func (p *DebianPackager) Init(ctx context.Context, ext pgxman.Extension, opts pgxman.PackagerOptions) error {
	workDir, buildDir, err := p.mkdir(opts)
	if err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	if err := p.downloadAndUnarchiveSource(ctx, ext, workDir, buildDir); err != nil {
		return fmt.Errorf("download and unarchive source: %w", err)
	}

	if err := p.installBuildDependencies(ctx, ext); err != nil {
		return fmt.Errorf("install build dependencies: %w", err)
	}

	if err := p.generateDebianTemplate(ext, buildDir); err != nil {
		return fmt.Errorf("generate debian template: %w", err)
	}

	return nil
}

func (p *DebianPackager) Pre(ctx context.Context, ext pgxman.Extension, opts pgxman.PackagerOptions) error {
	_, buildDir, err := p.mkdir(opts)
	if err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	return p.runScript(ctx, filepath.Join(buildDir, "script", "pre"), filepath.Join(buildDir, "src"))
}

func (p *DebianPackager) Post(ctx context.Context, ext pgxman.Extension, opts pgxman.PackagerOptions) error {
	_, buildDir, err := p.mkdir(opts)
	if err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	return p.runScript(ctx, filepath.Join(buildDir, "script", "post"), filepath.Join(buildDir, "src"))
}

// Package generates the following folder structure:
//
//   - workspace
//     -- extension.yaml
//     -- target
//     --- pgvector.orig.tar.gz
//     --- debian_build
//     ---- Makefile
//     ---- src
//     ---- debian
//     ---- script
func (p *DebianPackager) Main(ctx context.Context, ext pgxman.Extension, opts pgxman.PackagerOptions) error {
	_, buildDir, err := p.mkdir(opts)
	if err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	if err := p.buildDebian(ctx, ext, buildDir); err != nil {
		return fmt.Errorf("debian build: %w", err)
	}

	return nil
}

func (p *DebianPackager) downloadAndUnarchiveSource(ctx context.Context, ext pgxman.Extension, targetDir, buildDir string) error {
	sourceFile, err := p.copySource(ctx, ext, targetDir)
	if err != nil {
		return fmt.Errorf("download source %s: %w", ext.Source, err)
	}

	if err := p.unarchiveSource(ctx, sourceFile, buildDir); err != nil {
		return fmt.Errorf("unarchive source: %w", err)
	}

	return nil
}

func (p *DebianPackager) copySource(ctx context.Context, ext pgxman.Extension, targetDir string) (string, error) {
	logger := p.Logger.With(slog.String("source", ext.Source))
	logger.Info("Copying source")

	targetFile := filepath.Join(targetDir, fmt.Sprintf("%s_%s.orig.tar.gz", ext.Name, ext.Version))

	// file is already copied
	if _, err := os.Stat(targetFile); err == nil {
		return targetFile, nil
	}

	if err := cp.Copy(sourceTarGzPath, targetFile); err != nil {
		return "", err
	}

	return targetFile, nil
}

func (p *DebianPackager) unarchiveSource(ctx context.Context, sourceFile, buildDir string) error {
	logger := p.Logger.With(slog.String("file", sourceFile))
	logger.Info("Unarchiving source")

	sourceDir := filepath.Join(buildDir, "src")
	// file is already unarchived
	if _, err := os.Stat(sourceDir); err == nil {
		return nil
	}

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

func (p *DebianPackager) generateDebianTemplate(ext pgxman.Extension, buildDir string) error {
	logger := p.Logger.With(slog.String("name", ext.Name), slog.String("version", ext.Version))
	logger.Info("Generating debian package")

	return tmpl.Export(debian.FS, debianPackageTemplater{ext}, buildDir)
}

func (p *DebianPackager) installBuildDependencies(ctx context.Context, ext pgxman.Extension) error {
	builder := ext.Builders.Current()

	deps := ext.BuildDependencies
	if len(builder.BuildDependencies) > 0 {
		deps = builder.BuildDependencies
	}

	var depsToInstall []AptPackage
	for _, dep := range deps {
		if strings.Contains(dep, extensionDepPrefix) {
			dep = strings.TrimPrefix(dep, extensionDepPrefix)
			for _, ver := range ext.PGVersions {
				depsToInstall = append(
					depsToInstall,
					AptPackage{
						Pkg: extensionDebPkg(string(ver), dep),
					},
				)
			}
		} else {
			depsToInstall = append(
				depsToInstall,
				AptPackage{
					Pkg: dep,
				},
			)
		}
	}

	logger := p.Logger.With(slog.String("name", ext.Name), slog.String("version", ext.Version), slog.Any("deps", depsToInstall))
	logger.Info("Installing build deps")

	apt := &Apt{
		Logger: logger,
	}
	sources, err := apt.ConvertSources(ctx, builder.AptRepositories)
	if err != nil {
		return err
	}

	return apt.Install(ctx, depsToInstall, sources)
}

func (p *DebianPackager) runScript(ctx context.Context, script, sourceDir string) error {
	logger := p.Logger.With(slog.String("script", script))
	logger.Info("Running script")

	runScript := exec.CommandContext(ctx, "bash", script)
	runScript.Dir = sourceDir
	runScript.Stdout = os.Stdout
	runScript.Stderr = os.Stderr

	if err := runScript.Run(); err != nil {
		return fmt.Errorf("running script: %w", err)
	}

	return nil
}

func (p *DebianPackager) buildDebian(ctx context.Context, ext pgxman.Extension, buildDir string) error {
	logger := p.Logger.With(slog.String("name", ext.Name), slog.String("version", ext.Version))
	logger.Info("Building debian package")

	buildext := exec.CommandContext(ctx, "pg_buildext", "updatecontrol")
	buildext.Dir = buildDir
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
	debuild.Dir = buildDir
	debuild.Stdout = os.Stdout
	debuild.Stderr = os.Stderr

	if err := debuild.Run(); err != nil {
		return fmt.Errorf("debuild: %w", err)
	}

	return nil
}

func (p *DebianPackager) mkdir(opts pgxman.PackagerOptions) (workDir string, buildDir string, err error) {
	workDir = filepath.Join(opts.WorkDir, "target")
	buildDir = filepath.Join(workDir, "debian_build")

	err = os.MkdirAll(buildDir, 0755)
	if err != nil {
		return "", "", err
	}

	return workDir, buildDir, nil
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

func (e extensionData) BuildDeps() string {
	required := []string{
		"debhelper (>= 9)",
		"postgresql-server-dev-all (>= 158~)",
	}

	deps := e.BuildDependencies
	if builders := e.Builders; builders != nil {
		builder := builders.Current()
		if len(builder.BuildDependencies) != 0 {
			deps = builder.BuildDependencies
		}
	}

	return strings.Join(append(required, e.expandDeps(deps)...), ", ")
}

func (e extensionData) Deps() string {
	required := []string{
		"${shlibs:Depends}",
		"${misc:Depends}",
	}

	deps := e.RunDependencies
	if builders := e.Builders; builders != nil {
		builder := builders.Current()
		if len(builder.RunDependencies) != 0 {
			deps = builder.RunDependencies
		}
	}

	return strings.Join(append(required, e.expandDeps(deps)...), ", ")
}

func (e extensionData) MainBuildScript() string {
	return e.concatBuildScript(e.Build.Main)
}

func (e extensionData) PreBuildScript() string {
	return e.concatBuildScript(e.Build.Pre)
}

func (e extensionData) PostBuildScript() string {
	return e.concatBuildScript(e.Build.Post)
}

func (e extensionData) TimeNow() string {
	return time.Now().Format(time.RFC1123Z)
}

func (e extensionData) expandDeps(deps []string) []string {
	var expandedDeps []string
	for _, dep := range deps {
		if strings.HasPrefix(dep, extensionDepPrefix) {
			dep = strings.TrimPrefix(dep, extensionDepPrefix)
			expandedDeps = append(expandedDeps, extensionDebPkg("PGVERSION", dep))
		} else {
			expandedDeps = append(expandedDeps, dep)
		}
	}

	return expandedDeps
}

func (e extensionData) concatBuildScript(scripts []pgxman.BuildScript) string {
	var steps []string
	for _, s := range scripts {
		step := fmt.Sprintf("echo %q\n", s.Name)
		step += s.Run

		steps = append(steps, step)
	}

	return strings.Join(steps, "\n\n")
}

type debianPackageTemplater struct {
	ext pgxman.Extension
}

func (d debianPackageTemplater) Render(content []byte, out io.Writer) error {
	t, err := template.New("").Parse(string(content))
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	d.ext.Name = debNormalizedName(d.ext.Name)
	if err := t.Execute(out, extensionData{d.ext}); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	return nil
}

func extensionDebPkg(pgversion, extName string) string {
	return fmt.Sprintf("postgresql-%s-pgxman-%s", pgversion, debNormalizedName(extName))
}
