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

func (i *DebianInstaller) Install(ctx context.Context, extFiles []pgxman.PGXManfile, optFuncs ...pgxman.InstallerOptionsFunc) error {
	opts := pgxman.NewInstallerOptions(optFuncs)
	i.Logger.Debug("Installing extensions", "pgxman.yaml", extFiles, "options", opts)

	i.Logger.Debug("Fetching installable extensions")
	installableExts, err := buildkit.Extensions(ctx)
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
	for _, extFile := range extFiles {
		for _, extToInstall := range extFile.Extensions {
			if err := extToInstall.Validate(); err != nil {
				return err
			}

			if extToInstall.Path != "" {
				aptPkgs = append(
					aptPkgs,
					AptPackage{
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

				for _, pgv := range extFile.PGVersions {
					aptPkgs = append(
						aptPkgs,
						AptPackage{
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
	}

	if len(aptPkgs) == 0 {
		return nil
	}

	apt := &Apt{
		Logger: i.Logger.WithGroup("apt"),
	}
	aptSources, err := apt.GetChangedSources(ctx, aptRepos)
	if err != nil {
		return err
	}

	if !opts.IgnorePrompt {
		if err := promptInstall(aptPkgs, aptSources); err != nil {
			return err
		}
	}

	return apt.Install(ctx, aptPkgs, aptSources)
}

func promptInstall(debPkgs []AptPackage, sources []AptSource) error {
	out := []string{
		"The following Debian packages will be installed:",
	}
	for _, debPkg := range debPkgs {
		out = append(out, "  "+debPkg.Pkg)
	}

	if len(sources) > 0 {
		out = append(out, "The following Apt repositories will be added or updated:")
		for _, source := range sources {
			out = append(out, "  "+source.ID)
		}
	}

	out = append(out, "Do you want to continue? [Y/n] ")
	fmt.Print(strings.Join(out, "\n"))

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		switch strings.ToLower(scanner.Text()) {
		case "y", "yes":
			return nil
		default:
			return fmt.Errorf("installation aborted")
		}
	}

	return nil
}
