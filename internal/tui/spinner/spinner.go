package spinner

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
)

func New(disable bool) Spinner {
	if disable {
		return &stubSpinner{}
	}

	return &spinnerImpl{
		Spinner: spinner.New(spinner.CharSets[24], 100*time.Millisecond),
	}
}

type Spinner interface {
	Start()
	Stop()
	WithIndicator(string) Spinner
	WithDone(string) Spinner
}

type stubSpinner struct{}

func (s *stubSpinner) Start() {}
func (s *stubSpinner) Stop()  {}
func (s *stubSpinner) WithIndicator(string) Spinner {
	return s
}
func (s *stubSpinner) WithDone(string) Spinner {
	return s
}

type spinnerImpl struct {
	*spinner.Spinner
}

func (s *spinnerImpl) WithIndicator(indicator string) Spinner {
	s.Spinner.Suffix = fmt.Sprintf(" %s", indicator)
	return s
}

func (s *spinnerImpl) WithDone(done string) Spinner {
	s.Spinner.FinalMSG = done
	return s
}
