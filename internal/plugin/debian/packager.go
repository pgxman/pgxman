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
	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/log"
	tmpl "github.com/pgxman/pgxman/internal/template"
	"github.com/pgxman/pgxman/internal/template/debian"
	"github.com/pgxman/pgxman/internal/template/script"
	"golang.org/x/sync/errgroup"
)

const (
	extensionDepPrefix = "pgxman/"
)

type DebianPackager struct {
	Logger *log.Logger
}

// Init generates the following folder structure:
//
//   - workspace
//     -- extension.yaml
//     -- target
//     --- script
//     ---- pre
//     ---- post
//     --- 15
//     ---- pgvector.orig.tar.gz
//     ---- debian_build
//     ----- Makefile
//     ----- src
//     ----- debian
//     ----- script
//     ------ main
//     --- 14
//     ---- pgvector.orig.tar.gz
//     ---- debian_build
//     ----- Makefile
//     ----- src
//     ----- debian
//     ----- script
//     ------ main
func (p *DebianPackager) Init(ctx context.Context, ext pgxman.Extension, opts pgxman.PackagerOptions) error {
	p.Logger.Debug("Init step", "opts", opts, "name", ext.Name)

	if err := checkRootAccess(); err != nil {
		return err
	}

	if err := p.installBuildDependencies(ctx, ext); err != nil {
		return fmt.Errorf("install build dependencies: %w", err)
	}

	if err := p.generatePrePostScripts(ext, p.targetScriptDir(opts)); err != nil {
		return fmt.Errorf("write pre/post scripts: %w", err)
	}

	for _, pkg := range ext.Packages() {
		if err := p.prepareBuildDir(opts, pkg); err != nil {
			return fmt.Errorf("prepare build dir: %w", err)
		}
	}

	return nil
}

func (p *DebianPackager) Pre(ctx context.Context, ext pgxman.Extension, opts pgxman.PackagerOptions) error {
	p.Logger.Debug("Pre step", "opts", opts, "name", ext.Name)

	if err := checkRootAccess(); err != nil {
		return err
	}

	return p.runScript(ctx, filepath.Join(opts.WorkDir, "target", "script", "pre"))
}

func (p *DebianPackager) Post(ctx context.Context, ext pgxman.Extension, opts pgxman.PackagerOptions) error {
	p.Logger.Debug("Post step", "opts", opts, "name", ext.Name)

	if err := checkRootAccess(); err != nil {
		return err
	}

	return p.runScript(ctx, filepath.Join(opts.WorkDir, "target", "script", "post"))
}

func (p *DebianPackager) Main(ctx context.Context, ext pgxman.Extension, opts pgxman.PackagerOptions) error {
	p.Logger.Debug("Main step", "opts", opts, "name", ext.Name)

	if err := checkRootAccess(); err != nil {
		return err
	}

	token := opts.Parallel
	g, gctx := errgroup.WithContext(ctx)
	for _, pkg := range ext.Packages() {
		pkg := pkg

		g.Go(func() error {
			if err := p.buildDebian(gctx, pkg, p.targetDebianBuildDir(opts, pkg.PGVersion)); err != nil {
				return fmt.Errorf("debian build: %w", err)
			}

			return nil
		})
		token--

		// if token is used up, kick off builds & wait for them to finish
		if token == 0 {
			if err := g.Wait(); err != nil {
				return err
			}

			// reset
			g, gctx = errgroup.WithContext(ctx)
			token = opts.Parallel
		}
	}

	return g.Wait()
}

func (p *DebianPackager) prepareBuildDir(opts pgxman.PackagerOptions, pkg pgxman.ExtensionPackage) error {
	targetPgVerDir := p.targetPgVerDir(opts, pkg.PGVersion)
	if err := os.MkdirAll(targetPgVerDir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	debianBuildDir := p.targetDebianBuildDir(opts, pkg.PGVersion)
	if err := os.MkdirAll(debianBuildDir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	p.Logger.Debug("Preparing build dir", "target", targetPgVerDir, "name", pkg.Name, "pgVer", pkg.PGVersion)

	sourceFile, err := p.downloadSource(pkg, targetPgVerDir)
	if err != nil {
		return fmt.Errorf("download source %s: %w", pkg.Source, err)
	}

	if err := p.unarchiveSource(sourceFile, debianBuildDir); err != nil {
		return fmt.Errorf("unarchive source: %w", err)
	}

	if err := p.generateDebianTemplate(pkg, debianBuildDir); err != nil {
		return fmt.Errorf("generate debian template: %w", err)
	}

	return nil
}

func (p *DebianPackager) generatePrePostScripts(ext pgxman.Extension, scriptDir string) error {
	logger := p.Logger.With("name", ext.Name, "script-dir", scriptDir)
	logger.Info("Generating pre/post scripts")

	return tmpl.ExportFS(script.FS, scriptTemplater{ext}, scriptDir)
}

func (p *DebianPackager) targetDir(opts pgxman.PackagerOptions) string {
	return filepath.Join(opts.WorkDir, "target")
}

func (p *DebianPackager) targetScriptDir(opts pgxman.PackagerOptions) string {
	return filepath.Join(p.targetDir(opts), "script")
}

func (p *DebianPackager) targetPgVerDir(opts pgxman.PackagerOptions, pgVer pgxman.PGVersion) string {
	return filepath.Join(p.targetDir(opts), string(pgVer))
}

func (p *DebianPackager) targetDebianBuildDir(opts pgxman.PackagerOptions, pgVer pgxman.PGVersion) string {
	return filepath.Join(p.targetPgVerDir(opts, pgVer), "debian_build")
}

func (p *DebianPackager) downloadSource(ext pgxman.ExtensionPackage, targetDir string) (string, error) {
	logger := p.Logger.With(slog.String("source", ext.Source))
	logger.Info("Downloading source")

	targetFile := filepath.Join(targetDir, fmt.Sprintf("%s_%s.orig.tar.gz", ext.Name, ext.Version))

	// file is already downloaded
	if _, err := os.Stat(targetFile); err == nil {
		return targetFile, nil
	}

	source, err := ext.ParseSource()
	if err != nil {
		return "", nil
	}

	if err := source.Archive(targetFile); err != nil {
		return "", err
	}

	return targetFile, nil
}

func (p *DebianPackager) unarchiveSource(sourceFile, buildDir string) error {
	logger := p.Logger.With(slog.String("path", sourceFile))
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

func (p *DebianPackager) generateDebianTemplate(ext pgxman.ExtensionPackage, debianBuildDir string) error {
	logger := p.Logger.With("name", ext.Name, "debian-build-dir", debianBuildDir)
	logger.Info("Generating debian template")

	return tmpl.ExportFS(debian.FS, debianPackageTemplater{ext}, debianBuildDir)
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

	apt, err := NewApt(p.Logger.WithGroup("apt"))
	if err != nil {
		return err
	}

	repos, err := coreAptRepos()
	if err != nil {
		return err
	}

	sources, err := apt.ConvertSources(ctx, append(repos, builder.AptRepositories...))
	if err != nil {
		return err
	}

	return apt.Install(ctx, depsToInstall, sources)
}

func (p *DebianPackager) runScript(ctx context.Context, file string) error {
	logger := p.Logger.With(slog.String("script", file))
	logger.Info("Running script")

	lw := logger.Writer(slog.LevelDebug)

	runScript := exec.CommandContext(ctx, "bash", file)
	runScript.Dir = filepath.Dir(file)
	runScript.Stdout = lw
	runScript.Stderr = lw

	if err := runScript.Run(); err != nil {
		return fmt.Errorf("running script: %w", err)
	}

	return nil
}

func (p *DebianPackager) buildDebian(ctx context.Context, pkg pgxman.ExtensionPackage, buildDir string) error {
	logger := p.Logger.WithGroup(string(pkg.PGVersion))
	logger = logger.With("name", pkg.Name, "version", pkg.Version, "build-dir", buildDir)
	logger.Info("Building debian package")

	buildext := exec.CommandContext(ctx, "pg_buildext", "updatecontrol")
	buildext.Dir = buildDir
	buildext.Stdout = os.Stdout
	buildext.Stderr = os.Stderr

	logger.Info("Running pg_buildext updatecontrol", "cmd", buildext.String())
	if err := buildext.Run(); err != nil {
		return fmt.Errorf("pg_buildext updatecontrol: %w", err)
	}

	debuild := exec.CommandContext(
		ctx,
		"debuild",
		"--preserve-env",
		"--preserve-envvar", "PATH",
		"-uc", "-us", "-B", "--lintian-opts", "--profile", "debian", "--allow-root",
	)
	debuild.Env = append(
		os.Environ(),
		fmt.Sprintf("DEB_BUILD_OPTIONS=noautodbgsym parallel=%d", runtime.NumCPU()),
	)
	debuild.Dir = buildDir
	debuild.Stdout = os.Stdout
	debuild.Stderr = os.Stderr

	logger.Info("Running debuild", "cmd", debuild.String())
	if err := debuild.Run(); err != nil {
		return fmt.Errorf("debuild: %w", err)
	}

	return nil
}

type extensionData struct {
	pgxman.ExtensionPackage
}

func (e extensionData) Maintainers() string {
	var maintainers []string
	for _, m := range e.ExtensionPackage.Maintainers {
		maintainers = append(maintainers, fmt.Sprintf("%s <%s>", m.Name, m.Email))
	}

	return strings.Join(maintainers, ", ")
}

func (e extensionData) BuildDeps() string {
	required := []string{
		"debhelper",
		fmt.Sprintf("postgresql-server-dev-%s", e.PGVersion),
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
	return concatBuildScript(e.Build.Main)
}

func (e extensionData) TimeNow() string {
	return time.Now().Format(time.RFC1123Z)
}

func (e extensionData) expandDeps(deps []string) []string {
	var expandedDeps []string
	for _, dep := range deps {
		if strings.HasPrefix(dep, extensionDepPrefix) {
			dep = strings.TrimPrefix(dep, extensionDepPrefix)
			dep = extensionDebPkg("PGVERSION", dep)
		}

		expandedDeps = append(expandedDeps, dep)
	}

	return expandedDeps
}

func concatBuildScript(scripts []pgxman.BuildScript) string {
	var steps []string
	for _, s := range scripts {
		step := fmt.Sprintf("echo %q\n", s.Name)
		step += s.Run

		steps = append(steps, step)
	}

	return strings.Join(steps, "\n\n")
}

type debianPackageTemplater struct {
	ext pgxman.ExtensionPackage
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

type scriptData struct {
	pgxman.Extension
}

func (s scriptData) PreBuildScript() string {
	return concatBuildScript(s.Build.Pre)
}

func (s scriptData) PostBuildScript() string {
	return concatBuildScript(s.Build.Post)
}

type scriptTemplater struct {
	ext pgxman.Extension
}

func (s scriptTemplater) Render(content []byte, out io.Writer) error {
	t, err := template.New("").Parse(string(content))
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	if err := t.Execute(out, scriptData{s.ext}); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	return nil
}

func extensionDebPkg(pgversion, extName string) string {
	return fmt.Sprintf("postgresql-%s-pgxman-%s", pgversion, debNormalizedName(extName))
}
