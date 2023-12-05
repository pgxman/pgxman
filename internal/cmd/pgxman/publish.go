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

	var platforms []oapi.Platform
	for k, v := range map[oapi.PlatformOs]*pgxman.AptExtensionBuilder{
		oapi.DebianBookworm: ext.Builders.DebianBookworm,
		oapi.UbuntuJammy:    ext.Builders.UbuntuJammy,
	} {
		if v == nil {
			continue
		}

		var arches []oapi.Architecture
		for _, a := range ext.Arch {
			arches = append(arches, oapi.Architecture(a))
		}

		var pgVers []oapi.PgVersion
		for _, pgVer := range ext.PGVersions {
			pgVers = append(pgVers, convertPgVer(pgVer))
		}

		buildDeps := []oapi.Dependency{}
		if len(ext.BuildDependencies) > 0 {
			buildDeps = ext.BuildDependencies
		}
		if len(v.BuildDependencies) > 0 {
			buildDeps = v.BuildDependencies
		}

		runDeps := []oapi.Dependency{}
		if len(ext.RunDependencies) > 0 {
			runDeps = ext.RunDependencies
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

	return oapi.PublishExtension{
		Description: ext.Description,
		Homepage:    ext.Homepage,
		Readme:      ext.Readme,
		Keywords:    ext.Keywords,
		License:     ext.License,
		Maintainers: maintainers,
		Repository:  ext.Repository,
		MakeLatest:  flagPublishLatest,
		Source:      ext.Source,
		Name:        ext.Name,
		Version:     ext.Version,
		Platforms:   platforms,
	}
}

func convertPgVer(v pgxman.PGVersion) oapi.PgVersion {
	return oapi.PgVersion("pg_" + string(v))
}
