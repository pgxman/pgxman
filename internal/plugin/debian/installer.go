package debian

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/buildkit"
	"github.com/pgxman/pgxman/internal/log"
)

type DebianInstaller struct {
	Logger *log.Logger
}

func (i *DebianInstaller) Upgrade(ctx context.Context, f pgxman.PGXManfile, optFuncs ...pgxman.InstallerOptionsFunc) error {
	return i.installOrUpgrade(ctx, f, true, optFuncs...)
}

func (i *DebianInstaller) Install(ctx context.Context, f pgxman.PGXManfile, optFuncs ...pgxman.InstallerOptionsFunc) error {
	return i.installOrUpgrade(ctx, f, false, optFuncs...)
}

func (i DebianInstaller) installOrUpgrade(ctx context.Context, f pgxman.PGXManfile, upgrade bool, optFuncs ...pgxman.InstallerOptionsFunc) error {
	opts := pgxman.NewInstallerOptions(optFuncs)
	i.Logger.Debug("Installing extensions", "manifest", f, "options", opts)

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
	for _, extToInstall := range f.Extensions {
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
					Pkg:  fmt.Sprintf("postgresql-%s-pgxman-%s=%s", f.Postgres.Version, debNormalizedName(extToInstall.Name), extToInstall.Version),
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

	apt, err := NewApt(opts.Sudo, i.Logger.WithGroup("apt"))
	if err != nil {
		return err
	}

	aptSources, err := apt.GetChangedSources(ctx, aptRepos)
	if err != nil {
		return err
	}

	if !opts.IgnorePrompt {
		if err := promptInstallOrUpgrade(aptPkgs, aptSources, upgrade); err != nil {
			return err
		}
	}

	if upgrade {
		return apt.Upgrade(ctx, aptPkgs, aptSources)
	}

	return apt.Install(ctx, aptPkgs, aptSources)
}

func promptInstallOrUpgrade(debPkgs []AptPackage, sources []AptSource, upgrade bool) error {
	var (
		action   = "installed"
		abortMsg = "installation aborted"
	)
	if upgrade {
		action = "upgraded"
		abortMsg = "upgrade aborted"
	}

	out := []string{
		fmt.Sprintf("The following Debian packages will be %s:", action),
	}
	for _, debPkg := range debPkgs {
		out = append(out, "  "+debPkg.Pkg)
	}

	if len(sources) > 0 {
		out = append(out, "The following Apt repositories will be added or updated:")
		for _, source := range sources {
			out = append(out, "  "+source.Name)
		}
	}

	out = append(out, "Do you want to continue? [Y/n] ")
	fmt.Print(strings.Join(out, "\n"))

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		switch strings.ToLower(scanner.Text()) {
		case "y", "yes", "":
			return nil
		default:
			return fmt.Errorf(abortMsg)
		}
	}

	return nil
}
