package pgxman

import (
	"os"
	"path/filepath"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/cmd"
	"github.com/spf13/cobra"
)

var (
	flagBuildSet       map[string]string
	flagBuildNoCache   bool
	flagBuildCacheFrom []string
	flagBuildCacheTo   []string
)

func newBuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build extension according to the configuration file",
		RunE:  runBuild,
	}

	cmd.PersistentFlags().StringToStringVarP(&flagBuildSet, "set", "s", nil, "Override values in the extension.yaml file in the format of --set KEY=VALUE, e.g. --set version=1.0.0 --set arch=[amd64,arm64] --set pgVersions=[10,11,12]")
	cmd.PersistentFlags().BoolVar(&flagBuildNoCache, "no-cache", false, "Do not use cache when building the image. The value is passed to docker buildx build --no-cache.")
	cmd.PersistentFlags().StringArrayVar(&flagBuildCacheFrom, "cache-from", nil, "External cache sources. The value is passed to docker buildx build --cache-from.")
	cmd.PersistentFlags().StringArrayVar(&flagBuildCacheTo, "cache-to", nil, "Cache export destinations. The value is passed to docker buildx build --cache-to.")

	return cmd
}

func runBuild(c *cobra.Command, args []string) error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	overrides := cmd.ParseMapFlag(flagBuildSet)

	ext, err := pgxman.ReadExtension(filepath.Join(pwd, "extension.yaml"), overrides)
	if err != nil {
		return err
	}

	builder := pgxman.NewBuilder(
		pgxman.BuilderOptions{
			ExtDir:    pwd,
			Debug:     flagDebug,
			NoCache:   flagBuildNoCache,
			CacheFrom: flagBuildCacheFrom,
			CacheTo:   flagBuildCacheTo,
		},
	)
	return builder.Build(c.Context(), ext)
}
