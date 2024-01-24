package pgxman

import (
	"fmt"
	"strings"

	"github.com/oapi-codegen/runtime/types"
	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/log"
	"github.com/pgxman/pgxman/internal/registry"
	"github.com/pgxman/pgxman/oapi"
	"github.com/spf13/cobra"
)

var (
	flagPublishLatest bool
)

func newPublishCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "publish [BUILDKIT...]",
		Short:  "Publish extension to the registry",
		RunE:   runPublish,
		Args:   cobra.MinimumNArgs(1),
		Hidden: true,
	}

	cmd.PersistentFlags().BoolVar(&flagPublishLatest, "latest", false, "Make the published version the latest version")

	return cmd
}

func runPublish(c *cobra.Command, args []string) error {
	client, err := registry.NewClient(flagRegistryURL)
	if err != nil {
		return err
	}

	logger := log.NewTextLogger()
	for _, arg := range args {
		ext, err := pgxman.ReadExtension(arg, nil)
		if err != nil {
			return err
		}
		pext := convertPublishExtension(ext)

		logger.Debug("Publishing extension", "ext", pext)
		if err := client.PublishExtension(c.Context(), pext); err != nil {
			return err
		}

		fmt.Printf("Published %s.\n", ext.Name)
	}

	return nil
}

func convertPublishExtension(ext pgxman.Extension) oapi.PublishExtension {
	var maintainers []oapi.Maintainer
	for _, m := range ext.Maintainers {
		maintainers = append(maintainers, oapi.Maintainer{
			Name:  m.Name,
			Email: types.Email(m.Email),
		})
	}

	pkgs := make(oapi.Packages)
	for _, pkg := range ext.Packages() {
		pkgs[string(pkg.PGVersion)] = oapi.Package{
			Description: pkg.Description,
			Homepage:    pkg.Homepage,
			License:     pkg.License,
			Maintainers: maintainers,
			Platforms:   convertPlatform(pkg),
			Readme:      pkg.Readme,
			Repository:  pkg.Repository,
			Source:      pkg.Source,
			Version:     pkg.Version,
		}
	}

	return oapi.PublishExtension{
		Keywords:   ext.Keywords,
		MakeLatest: flagPublishLatest,
		Name:       ext.Name,
		Packages:   pkgs,
	}
}

func convertPlatform(pkg pgxman.ExtensionPackage) []oapi.Platform {
	var platforms []oapi.Platform
	for k, v := range map[oapi.PlatformOs]*pgxman.AptExtensionBuilder{
		oapi.DebianBookworm: pkg.Builders.DebianBookworm,
		oapi.UbuntuJammy:    pkg.Builders.UbuntuJammy,
	} {
		if v == nil {
			continue
		}

		var arches []oapi.Architecture
		for _, a := range pkg.Arch {
			arches = append(arches, oapi.Architecture(a))
		}

		pgVers := []oapi.PgVersion{oapi.PgVersion(pkg.PGVersion)}

		buildDeps := []oapi.Dependency{}
		if len(pkg.BuildDependencies) > 0 {
			buildDeps = pkg.BuildDependencies
		}
		if len(v.BuildDependencies) > 0 {
			buildDeps = v.BuildDependencies
		}

		runDeps := []oapi.Dependency{}
		if len(pkg.RunDependencies) > 0 {
			runDeps = pkg.RunDependencies
		}
		if len(v.RunDependencies) > 0 {
			runDeps = v.RunDependencies
		}

		var aptRepos []oapi.AptRepository
		for _, r := range v.AptRepositories {
			var types []oapi.AptRepositoryType
			for _, t := range r.Types {
				types = append(types, oapi.AptRepositoryType(strings.ReplaceAll(string(t), "-", "_")))
			}

			aptRepos = append(aptRepos, oapi.AptRepository{
				Components: r.Components,
				Id:         r.ID,
				SignedKey: oapi.SignedKey{
					Url:    r.SignedKey.URL,
					Format: oapi.SignedKeyFormat(r.SignedKey.Format),
				},
				Suites: r.Suites,
				Types:  types,
				Uris:   r.URIs,
			})
		}

		platforms = append(platforms, oapi.Platform{
			Os:                k,
			Architectures:     arches,
			PgVersions:        pgVers,
			BuildDependencies: buildDeps,
			RunDependencies:   runDeps,
			AptRepositories:   aptRepos,
		})
	}

	return platforms
}
