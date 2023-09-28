package errorsx

import (
	"errors"
	"fmt"

	"github.com/pgxman/pgxman/internal/plugin"
)

var (
	errCommandNotSupported = fmt.Errorf("this command is not currently supported on your OS")
)

func Pretty(err error) error {
	if e := new(plugin.ErrUnsupportedPlugin); errors.As(err, &e) {
		return errCommandNotSupported
	}

	return err
}
