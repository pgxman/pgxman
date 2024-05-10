package upgrade

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/google/go-github/v57/github"
	"github.com/pgxman/pgxman/internal/config"
	"github.com/pgxman/pgxman/internal/log"
	"github.com/stretchr/testify/assert"
)

func Test_Checker(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `
{
	"tag_name": "v1.0.1"
}

		`)
	}))
	defer ts.Close()

	url, _ := url.Parse(ts.URL + "/")
	ghClient := github.NewClient(nil)
	ghClient.BaseURL = url

	cases := []struct {
		Name               string
		CurrentVersion     string
		LastCheckTime      time.Time
		WantHasUpgrade     bool
		WantCurrentVersion *semver.Version
		WantLatestVersion  *semver.Version
	}{
		{
			Name:               "has upgrade",
			CurrentVersion:     "v1.0.0",
			LastCheckTime:      time.Now().Add(-checkInterval),
			WantHasUpgrade:     true,
			WantCurrentVersion: semver.MustParse("1.0.0"),
			WantLatestVersion:  semver.MustParse("1.0.1"),
		},
		{
			Name:               "no upgrade",
			CurrentVersion:     "v1.0.2",
			LastCheckTime:      time.Now().Add(-checkInterval),
			WantHasUpgrade:     false,
			WantCurrentVersion: semver.MustParse("1.0.2"),
			WantLatestVersion:  semver.MustParse("1.0.1"),
		},
		{
			Name:               "skip upgrade for dev",
			CurrentVersion:     "dev",
			LastCheckTime:      time.Now().Add(-checkInterval),
			WantHasUpgrade:     false,
			WantCurrentVersion: nil,
			WantLatestVersion:  nil,
		},
		{
			Name:               "skip checking upgrade",
			CurrentVersion:     "v1.0.0",
			LastCheckTime:      time.Now(),
			WantHasUpgrade:     false,
			WantCurrentVersion: nil,
			WantLatestVersion:  nil,
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.Name, func(t *testing.T) {
			assert := assert.New(t)

			checker := Checker{
				ghClient:       ghClient,
				logger:         log.NewTextLogger(),
				currentVersion: c.CurrentVersion,
				readConfigFunc: func() (*config.Config, error) {
					return &config.Config{
						LastUpgradeCheckTime: c.LastCheckTime,
					}, nil
				},
				writeConfigFunc: func(c config.Config) error {
					assert.WithinDuration(c.LastUpgradeCheckTime, time.Now(), 2*time.Second)
					return nil
				},
				enabled: true,
			}

			result, err := checker.Check(context.TODO())
			assert.NoError(err)
			assert.Equal(c.WantHasUpgrade, result.HasUpgrade)
			assert.Equal(c.WantCurrentVersion, result.CurrentVersion)
			assert.Equal(c.WantLatestVersion, result.LatestVersion)
		})
	}
}
