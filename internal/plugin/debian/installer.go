package debian

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"log/slog"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/buildkit"
	"github.com/pgxman/pgxman/internal/log"
)

type DebianInstaller struct {
	Logger *log.Logger
}

type installDebPkg struct {
	Pkg  string
	Opts []string
}

func (i *DebianInstaller) Install(ctx context.Context, extFiles []pgxman.PGXManfile, optFuncs ...pgxman.InstallerOptionsFunc) error {
	opts := pgxman.NewInstallerOptions(optFuncs)
	i.Logger.Debug("Installing extensions", "pgxman.yaml", extFiles, "options", opts)

	i.Logger.Debug("Fetching installable extensions")
	installableExts, err := buildkit.Extensions()
	if err != nil {
		return fmt.Errorf("fetch installable extensions: %w", err)
	}

	var (
		debPkgs  []installDebPkg
		aptRepos []pgxman.AptRepository
	)
	for _, extFile := range extFiles {
		for _, extToInstall := range extFile.Extensions {
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

				for _, pgv := range extFile.PGVersions {
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
	}

	if len(debPkgs) == 0 {
		return nil
	}

	if !opts.IgnorePrompt {
		if err := promptInstall(debPkgs, aptRepos); err != nil {
			return err
		}
	}

	i.Logger.Debug("Installing debian packages", "packages", debPkgs)
	return runAptInstall(ctx, debPkgs, aptRepos, i.Logger)
}

func promptInstall(debPkgs []installDebPkg, aptRepos []pgxman.AptRepository) error {
	out := []string{
		"The following Debian packages will be installed:",
	}
	for _, debPkg := range debPkgs {
		out = append(out, "  "+debPkg.Pkg)
	}

	if len(aptRepos) > 0 {
		out = append(out, "The following Apt repositories will be added:")
		for _, ar := range aptRepos {
			out = append(out, "  "+ar.ID)
		}
	}

	out = append(out, "Do you want to continue? [Y/n] ")
	fmt.Print(strings.Join(out, "\n"))

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		switch scanner.Text() {
		case "Y", "y":
			return nil
		default:
			return fmt.Errorf("installation aborted")
		}
	}

	return nil
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
