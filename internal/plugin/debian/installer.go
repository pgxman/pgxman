package debian

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/buildkit"
	"github.com/pgxman/pgxman/internal/log"
	"golang.org/x/exp/slog"
)

type DebianInstaller struct {
	Logger *log.Logger
}

type installDebPkg struct {
	Pkg  string
	Opts []string
}

func (i *DebianInstaller) Install(ctx context.Context, exts pgxman.PGXManfile) error {
	i.Logger.Debug("Installing extensions", "pgxman.yaml", exts)

	i.Logger.Debug("Fetching installable extensions")
	installableExts, err := buildkit.InstallableExtensions(ctx)
	if err != nil {
		return fmt.Errorf("fetch installable extensions: %w", err)
	}

	var (
		debPkgs  []installDebPkg
		aptRepos []pgxman.AptRepository
	)
	for _, extToInstall := range exts.Extensions {
		if err := extToInstall.Validate(); err != nil {
			return err
		}

		if extToInstall.Path != "" {
			debPkgs = append(
				debPkgs,
				installDebPkg{
					Pkg:  extToInstall.Path,
					Opts: extToInstall.Options,
				},
			)
		} else {
			installableExt, ok := installableExts[extToInstall.Name]
			if !ok {
				return fmt.Errorf("extension %q not found", extToInstall.Name)
			}
			if installableExt.Version != extToInstall.Version {
				return fmt.Errorf("extension %q with version %q not available", extToInstall.Name, extToInstall.Version)
			}

			for _, pgv := range exts.PGVersions {
				debPkgs = append(
					debPkgs,
					installDebPkg{
						Pkg:  fmt.Sprintf("postgresql-%s-pgxman-%s=%s", pgv, debNormalizedName(extToInstall.Name), extToInstall.Version),
						Opts: extToInstall.Options,
					},
				)
			}

			if builders := installableExt.Builders; builders != nil {
				builder := builders.Current()
				if ar := builder.AptRepositories; len(ar) > 0 {
					aptRepos = append(aptRepos, ar...)
				}
			}
		}
	}

	i.Logger.Debug("Installing debian packages", "packages", debPkgs)
	return runAptInstall(ctx, debPkgs, aptRepos, i.Logger)
}

func runAptInstall(ctx context.Context, debPkgs []installDebPkg, aptRepos []pgxman.AptRepository, logger *log.Logger) error {
	logger.Debug("add apt repo", slog.Any("repos", aptRepos))
	if err := addAptRepos(ctx, aptRepos, logger); err != nil {
		return fmt.Errorf("add apt repos: %w", err)
	}

	for _, pkg := range debPkgs {
		logger.Debug("Running apt install", "package", pkg)

		opts := []string{"install", "-y", "--no-install-recommends"}
		opts = append(opts, pkg.Opts...)
		opts = append(opts, pkg.Pkg)

		cmd := exec.CommandContext(ctx, "apt", opts...)
		cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("apt install: %w", err)
		}
	}

	return nil
}
