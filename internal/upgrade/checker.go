package upgrade

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/google/go-github/v57/github"
	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/config"
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
	}
}

type CheckResult struct {
	HasUpgrade     bool
	CurrentVersion *semver.Version
	LatestVersion  *semver.Version
}

type Checker struct {
	ghClient        *github.Client
	logger          *log.Logger
	currentVersion  string
	readConfigFunc  func() (*config.Config, error)
	writeConfigFunc func(c config.Config) error
}

func (c *Checker) Check(ctx context.Context) (result *CheckResult, err error) {
	if c.currentVersion == "dev" {
		c.logger.Debug("disabled upgrade checking")
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
		c.logger.Debug("skip checking upgrade", "latst", lastCheckTime, "next", nextCheckTime)
		return &CheckResult{
			HasUpgrade: false,
		}, nil
	}

	rel, _, err := c.ghClient.Repositories.GetLatestRelease(ctx, "pgxman", "pgxman")
	if err != nil {
		return nil, err
	}

	latestVer, err := parseSemVar(*rel.TagName)
	if err != nil {
		return nil, fmt.Errorf("error parsing tag name as version: %w", err)
	}

	currVer, err := parseSemVar(c.currentVersion)
	if err != nil {
		return nil, fmt.Errorf("error parsing version: %w", err)
	}

	var hasUpgrade bool
	if currVer.LessThan(latestVer) {
		hasUpgrade = true
	}

	return &CheckResult{
		HasUpgrade:     hasUpgrade,
		CurrentVersion: currVer,
		LatestVersion:  latestVer,
	}, nil
}

func parseSemVar(v string) (*semver.Version, error) {
	return semver.StrictNewVersion(strings.TrimPrefix(v, "v"))
}
