package e2etest

import (
	"flag"
	"os"
	"os/signal"
	"sync"
	"testing"

	"github.com/pgxman/pgxman/internal/log"
	"golang.org/x/exp/slog"
)

var (
	flagBuildImage string
)

var cf struct {
	o sync.Once
	m sync.RWMutex
	f []func()
}

// cleanupOnInterrupt registers a signal handler and will execute a stack of functions if an interrupt signal is caught
func cleanupOnInterrupt(c chan os.Signal) {
	for range c {
		cf.o.Do(func() {
			cf.m.RLock()
			defer cf.m.RUnlock()
			for i := len(cf.f) - 1; i >= 0; i-- {
				cf.f[i]()
			}
			os.Exit(1)
		})
	}
}

// addCleanupOnInterrupt stores cleanup functions to execute if an interrupt signal is caught
func addCleanupOnInterrupt(cleanup func()) {
	cf.m.Lock()
	defer cf.m.Unlock()
	cf.f = append(cf.f, cleanup)
}

// EnsureCleanup will run the provided cleanup function when the test ends,
// either via t.Cleanup or on interrupt via CleanupOnInterrupt.
func EnsureCleanup(t *testing.T, cleanup func()) {
	t.Cleanup(cleanup)
	addCleanupOnInterrupt(cleanup)
}

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

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go cleanupOnInterrupt(c)

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
