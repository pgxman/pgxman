package debian

import (
	"context"
	"fmt"
	"os"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/buildkit"
	"github.com/pgxman/pgxman/internal/log"
)

type DebianInstaller struct {
	Logger *log.Logger
}

func (i *DebianInstaller) Upgrade(ctx context.Context, b pgxman.Bundle, optFuncs ...pgxman.InstallerOptionsFunc) error {
	return i.installOrUpgrade(ctx, b, true, optFuncs...)
}

func (i *DebianInstaller) Install(ctx context.Context, b pgxman.Bundle, optFuncs ...pgxman.InstallerOptionsFunc) error {
	return i.installOrUpgrade(ctx, b, false, optFuncs...)
}

func (i DebianInstaller) installOrUpgrade(ctx context.Context, bundle pgxman.Bundle, upgrade bool, optFuncs ...pgxman.InstallerOptionsFunc) error {
	opts := pgxman.NewInstallerOptions(optFuncs)
	i.Logger.Debug("Installing extensions", "bundle", bundle, "options", opts)

	if err := checkRootAccess(); err != nil {
		return err
	}

	i.Logger.Debug("Fetching installable extensions")
	installableExts, err := buildkit.Extensions()
	if err != nil {
		return fmt.Errorf("fetch installable extensions: %w", err)
	}

	aptRepos, err := coreAptRepos()
	if err != nil {
		return err
	}

	var (
		aptPkgs []AptPackage
	)
	for _, extToInstall := range bundle.Extensions {
		if err := extToInstall.Validate(); err != nil {
			return err
		}

		if extToInstall.Path != "" {
			aptPkgs = append(
				aptPkgs,
				AptPackage{
					Pkg:     extToInstall.Path,
					IsLocal: true,
					Opts:    extToInstall.Options,
				},
			)
		} else {
			installableExt, ok := installableExts[extToInstall.Name]
			if !ok {
				return fmt.Errorf("extension %q not found", extToInstall.Name)
			}

			aptPkgs = append(
				aptPkgs,
				AptPackage{
					Pkg:  fmt.Sprintf("postgresql-%s-pgxman-%s=%s", bundle.Postgres.Version, debNormalizedName(extToInstall.Name), extToInstall.Version),
					Opts: extToInstall.Options,
				},
			)

			if builders := installableExt.Builders; builders != nil {
				builder := builders.Current()
				if ar := builder.AptRepositories; len(ar) > 0 {
					aptRepos = append(aptRepos, ar...)
				}
			}
		}
	}

	if len(aptPkgs) == 0 {
		return nil
	}

	apt, err := NewApt(i.Logger.WithGroup("apt"))
	if err != nil {
		return err
	}

	aptSources, err := apt.GetChangedSources(ctx, aptRepos)
	if err != nil {
		return err
	}

	if h := opts.BeforeRunHook; h != nil {
		var pkgs []string
		for _, pkg := range aptPkgs {
			pkgs = append(pkgs, pkg.Pkg)
		}
		var sources []string
		for _, source := range aptSources {
			sources = append(sources, source.Name)
		}
		if err := h(pkgs, sources); err != nil {
			return err
		}
	}

	if upgrade {
		return apt.Upgrade(ctx, aptPkgs, aptSources)
	}

	return apt.Install(ctx, aptPkgs, aptSources)
}

func checkRootAccess() error {
	if os.Getuid() != 0 {
		return pgxman.ErrRootAccessRequired
	}
	return nil
}
