package upgrade

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/google/go-github/v57/github"
	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/config"
	"github.com/pgxman/pgxman/internal/iostreams"
	"github.com/pgxman/pgxman/internal/log"
)

const (
	checkInterval = 24 * time.Hour
)

func NewChecker(logger *log.Logger) *Checker {
	httpClient := &http.Client{
		Timeout: 5 * time.Second,
	}
	return &Checker{
		ghClient:        github.NewClient(httpClient),
		logger:          logger,
		currentVersion:  pgxman.Version,
		readConfigFunc:  config.Read,
		writeConfigFunc: config.Write,
		enabled:         shouldEnable(pgxman.Version),
	}
}

type CheckResult struct {
	CurrentVersion *semver.Version
	LatestVersion  *semver.Version
	HasUpgrade     bool
}

type Checker struct {
	ghClient        *github.Client
	logger          *log.Logger
	readConfigFunc  func() (*config.Config, error)
	writeConfigFunc func(c config.Config) error
	currentVersion  string
	enabled         bool
}

func (c *Checker) Check(ctx context.Context) (result *CheckResult, err error) {
	logger := c.logger.With("current", c.currentVersion)

	if !c.enabled {
		logger.Debug("disabled upgrade checking")
		return &CheckResult{
			HasUpgrade: false,
		}, nil
	}

	var (
		lastCheckTime time.Time
		now           = time.Now()
	)
	cfg, err := c.readConfigFunc()
	if err == nil {
		lastCheckTime = cfg.LastUpgradeCheckTime
		cfg.LastUpgradeCheckTime = now
	} else {
		cfg = &config.Config{
			LastUpgradeCheckTime: now,
		}
	}

	defer func() {
		err = errors.Join(err, c.writeConfigFunc(*cfg))
	}()

	nextCheckTime := lastCheckTime.Add(checkInterval)
	if now.Before(nextCheckTime) {
		logger.Debug("skip checking upgrade", "latst", lastCheckTime, "next", nextCheckTime)
		return &CheckResult{
			HasUpgrade: false,
		}, nil
	}

	rel, _, err := c.ghClient.Repositories.GetLatestRelease(ctx, "pgxman", "pgxman")
	if err != nil {
		logger.Debug("error getting latest release", "error", err)
		return &CheckResult{
			HasUpgrade: false,
		}, nil
	}

	currVer, err := parseSemVar(c.currentVersion)
	if err != nil {
		logger.Debug("error parsing current version")
		return &CheckResult{
			HasUpgrade: false,
		}, nil
	}

	latestVer, err := parseSemVar(*rel.TagName)
	if err != nil {
		logger.Debug("error parsing tag name as version", "error", err)
		return &CheckResult{
			HasUpgrade: false,
		}, nil
	}

	return &CheckResult{
		HasUpgrade:     currVer.LessThan(latestVer),
		CurrentVersion: currVer,
		LatestVersion:  latestVer,
	}, nil
}

func parseSemVar(v string) (*semver.Version, error) {
	return semver.StrictNewVersion(strings.TrimPrefix(v, "v"))
}

func shouldEnable(currentVersion string) bool {
	if os.Getenv("PGXMAN_NO_UPGRADE_NOTIFIER") != "" {
		return false
	}

	if currentVersion == "dev" {
		return false
	}

	return pgxman.UpdaterEnabled == "true" &&
		!isCI() &&
		iostreams.IsTerminal(os.Stdout) &&
		iostreams.IsTerminal(os.Stderr)
}

// based on https://github.com/watson/ci-info/blob/HEAD/index.js
func isCI() bool {
	return os.Getenv("CI") != "" || // GitHub Actions, Travis CI, CircleCI, Cirrus CI, GitLab CI, AppVeyor, CodeShip, dsari
		os.Getenv("BUILD_NUMBER") != "" || // Jenkins, TeamCity
		os.Getenv("RUN_ID") != "" // TaskCluster, dsari
}
