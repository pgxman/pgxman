package e2etest

import (
	"flag"
	"os"
	"testing"

	"github.com/pgxman/pgxman/internal/log"
	"golang.org/x/exp/slog"
)

var (
	flagBuildImage string
)

func TestMain(m *testing.M) {
	var e2etest bool
	flag.BoolVar(&e2etest, "e2e", false, "Run e2e tests")
	flag.StringVar(&flagBuildImage, "build-image", "", "Build image")
	flag.Parse()

	log.SetLevel(slog.LevelDebug)
	logger := log.NewTextLogger()

	if !e2etest {
		logger.Info("e2e tests are skipped")
		os.Exit(0)
	}

	if flagBuildImage == "" {
		logger.Info("-build-image is required")
		os.Exit(0)
	}

	os.Exit(m.Run())
}

type Step struct {
	Name string
	Step func(t *testing.T)
}

func RunSteps(t *testing.T, steps []Step) {
	for i, s := range steps {
		ii, ss := i, s
		if ii == 0 {
			t.Run(ss.Name, ss.Step)
		} else {
			t.Run(ss.Name, func(tt *testing.T) {
				if t.Failed() {
					tt.SkipNow()
				}
				ss.Step(tt)
			})
		}
	}
}
