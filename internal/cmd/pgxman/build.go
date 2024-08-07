package pgxman

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/cmd"
	"github.com/spf13/cobra"
)

var (
	flagBuildSet           map[string]string
	flagBuildExtensionFile string
	flagBuildNoCache       bool
	flagBuildParallel      int
	flagBuildCacheFrom     []string
	flagBuildCacheTo       []string
	flagBuildPull          bool
)

func newBuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "build",
		Aliases: []string{"b"},
		Short:   "Build extension according to the configuration file",
		RunE:    runBuild,
	}

	cmd.PersistentFlags().StringVarP(&flagBuildExtensionFile, "file", "f", "extension.yaml", "Path to the extension manifest file")
	cmd.PersistentFlags().StringToStringVarP(&flagBuildSet, "set", "s", nil, "Override values in the extension.yaml file in the format of --set KEY=VALUE, e.g. --set version=1.0.0 --set arch=[amd64,arm64] --set pgVersions=[10,11,12]")
	cmd.PersistentFlags().BoolVar(&flagBuildNoCache, "no-cache", false, "Do not use cache when building the image. The value is passed to docker buildx build --no-cache.")
	cmd.PersistentFlags().StringArrayVar(&flagBuildCacheFrom, "cache-from", nil, "External cache sources. The value is passed to docker buildx build --cache-from.")
	cmd.PersistentFlags().StringArrayVar(&flagBuildCacheTo, "cache-to", nil, "Cache export destinations. The value is passed to docker buildx build --cache-to.")
	cmd.PersistentFlags().IntVar(&flagBuildParallel, "parallel", 2, "Number of parrallel builds to run")
	cmd.PersistentFlags().BoolVar(&flagBuildPull, "pull", false, "Always attempt to pull all referenced images")

	return cmd
}

func runBuild(c *cobra.Command, args []string) error {
	if flagBuildParallel < 1 {
		return fmt.Errorf("invalid parallel value: %d", flagBuildParallel)
	}

	extFile, err := filepath.Abs(flagBuildExtensionFile)
	if err != nil {
		return err
	}

	overrides := cmd.ParseMapFlag(flagBuildSet)
	ext, err := pgxman.ReadExtension(extFile, overrides)
	if err != nil {
		return err
	}

	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	builder := pgxman.NewBuilder(
		pgxman.BuilderOptions{
			ExtDir:    pwd,
			Debug:     flagDebug,
			Parallel:  flagBuildParallel,
			NoCache:   flagBuildNoCache,
			CacheFrom: flagBuildCacheFrom,
			CacheTo:   flagBuildCacheTo,
			Pull:      flagBuildPull,
		},
	)
	return builder.Build(c.Context(), ext)
}
