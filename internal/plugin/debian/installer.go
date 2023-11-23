package debian

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/eiannone/keyboard"
	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/buildkit"
	"github.com/pgxman/pgxman/internal/log"
)

type DebianInstaller struct {
	Logger *log.Logger
}

func (i *DebianInstaller) Install(ctx context.Context, ext pgxman.InstallExtension) error {
	return i.installOrUpgrade(ctx, ext, false)
}

func (i *DebianInstaller) Upgrade(ctx context.Context, ext pgxman.InstallExtension) error {
	return i.installOrUpgrade(ctx, ext, true)
}

func (i *DebianInstaller) PreInstallCheck(ctx context.Context, exts []pgxman.InstallExtension, io pgxman.IO) error {
	return i.installOrUpgradeCheck(ctx, exts, io, false)
}

func (i *DebianInstaller) PreUpgradeCheck(ctx context.Context, exts []pgxman.InstallExtension, io pgxman.IO) error {
	return i.installOrUpgradeCheck(ctx, exts, io, true)
}

func (i DebianInstaller) installOrUpgradeCheck(ctx context.Context, exts []pgxman.InstallExtension, io pgxman.IO, upgrade bool) error {
	if err := checkRootAccess(); err != nil {
		return err
	}

	var (
		aptPkgs  []AptPackage
		aptRepos []pgxman.AptRepository
	)
	for _, extToInstall := range exts {
		if err := extToInstall.Validate(); err != nil {
			return err
		}

		aptPkg, err := newAptPackage(extToInstall)
		if err != nil {
			return err
		}

		aptPkgs = append(aptPkgs, aptPkg)
		aptRepos = append(aptRepos, aptPkg.Repos...)
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

	return promptInstallOrUpgrade(io, aptPkgs, aptSources, upgrade)
}

func (i DebianInstaller) installOrUpgrade(ctx context.Context, ext pgxman.InstallExtension, upgrade bool) error {
	i.Logger.Debug("Installing extension", "extension", ext)

	if err := checkRootAccess(); err != nil {
		return err
	}

	if err := ext.Validate(); err != nil {
		return err
	}

	aptPkg, err := newAptPackage(ext)
	if err != nil {
		return err
	}

	apt, err := NewApt(i.Logger.WithGroup("apt"))
	if err != nil {
		return err
	}

	aptSources, err := apt.GetChangedSources(ctx, aptPkg.Repos)
	if err != nil {
		return err
	}

	if upgrade {
		return apt.Upgrade(ctx, []AptPackage{aptPkg}, aptSources)
	}

	return apt.Install(ctx, []AptPackage{aptPkg}, aptSources)
}

func extDebPkgName(ext pgxman.InstallExtension) string {
	return fmt.Sprintf("postgresql-%s-pgxman-%s=%s", ext.PGVersion, debNormalizedName(ext.Name), ext.Version)
}

func checkRootAccess() error {
	if os.Getuid() != 0 {
		return pgxman.ErrRootAccessRequired
	}
	return nil
}

func newAptPackage(ext pgxman.InstallExtension) (AptPackage, error) {
	var aptPkg AptPackage

	installableExts, err := buildkit.Extensions()
	if err != nil {
		return aptPkg, fmt.Errorf("fetch installable extensions: %w", err)
	}

	coreAptRepos, err := coreAptRepos()
	if err != nil {
		return aptPkg, err
	}

	if ext.Path != "" {
		aptPkg = AptPackage{
			Pkg:     ext.Path,
			IsLocal: true,
			Opts:    ext.Options,
		}
	} else {
		installableExt, ok := installableExts[ext.Name]
		if !ok {
			return aptPkg, fmt.Errorf("extension %q not found", ext.Name)
		}

		aptPkg = AptPackage{
			Pkg:   extDebPkgName(ext),
			Opts:  ext.Options,
			Repos: coreAptRepos,
		}

		if builders := installableExt.Builders; builders != nil {
			builder := builders.Current()
			if ar := builder.AptRepositories; len(ar) > 0 {
				aptPkg.Repos = append(aptPkg.Repos, ar...)
			}
		}
	}

	return aptPkg, nil
}

func promptInstallOrUpgrade(io pgxman.IO, debPkgs []AptPackage, sources []AptSource, upgrade bool) error {
	if !io.IsTerminal() {
		return nil
	}

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
	fmt.Fprint(io.Stdout, strings.Join(out, "\n"))

	if err := keyboard.Open(); err != nil {
		return err
	}
	defer keyboard.Close()

	for {
		char, key, err := keyboard.GetKey()
		if err != nil {
			return err
		}

		if char == 'y' || char == 'Y' || key == keyboard.KeyEnter {
			fmt.Println()
			break
		} else {
			fmt.Println()
			return fmt.Errorf(abortMsg)
		}
	}

	return nil
}
