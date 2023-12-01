package doctor

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"

	"github.com/pgxman/pgxman/internal/docker"
	"github.com/pgxman/pgxman/internal/pg"
)

func Validate(ctx context.Context) (required []ValidationResult, optional []ValidationResult) {
	var (
		validators []Validator
	)

	if runtime.GOOS == "linux" {
		validators = append(validators, &postgresValidator{})
	}
	validators = append(validators, &dockerValidator{})

	for _, v := range validators {
		for _, r := range v.Validate(ctx) {
			switch r.Category {
			case ValidationCategoryRequired:
				required = append(required, r)
			case ValidationCategoryOptional:
				optional = append(optional, r)
			default:
				panic(fmt.Sprintf("unknown validation result category: %s", r.Category))
			}
		}
	}

	return required, optional
}

const (
	ValidationSuccess ValidationResultType = "success"
	ValidationWarning ValidationResultType = "warning"
	ValiationError    ValidationResultType = "error"
)

type ValidationResultType string

type ValidationResult struct {
	Type     ValidationResultType
	Category ValidationResultCategory
	Message  string
}

const (
	ValidationCategoryRequired ValidationResultCategory = "required"
	ValidationCategoryOptional ValidationResultCategory = "optional"
)

type ValidationResultCategory string

type Validator interface {
	Validate(context.Context) []ValidationResult
}

type dockerValidator struct {
}

func (v *dockerValidator) Validate(ctx context.Context) []ValidationResult {
	var (
		results  []ValidationResult
		category ValidationResultCategory
	)

	switch runtime.GOOS {
	case "linux":
		category = ValidationCategoryOptional
	case "darwin":
		category = ValidationCategoryRequired
	default:
		category = ValidationCategoryRequired
	}

	var (
		dockerIsInstalled = ValidationResult{
			Type:     ValidationSuccess,
			Category: category,
			Message:  "Docker is installed",
		}
		dockerIsRunning = ValidationResult{
			Type:     ValidationSuccess,
			Category: category,
			Message:  `Docker daemon is running`,
		}
	)

	dockerErr := docker.CheckInstall(ctx)
	if dockerErr != nil {
		if errors.Is(dockerErr, docker.ErrClientNotFound) {
			var (
				lines      []string
				resultType ValidationResultType
			)
			if runtime.GOOS == "linux" {
				lines = []string{
					"To use the `pgxman container` commands, you'll need to install Docker.",
					"Visit https://docs.docker.com/engine/install for more info.",
				}
				resultType = ValidationWarning
			} else if runtime.GOOS == "darwin" {
				lines = []string{
					"pgxman emulates the production experience on macOS.",
					"To use the `pgxman install` & `pgxman container` commands, you'll need to install Docker.",
					"Visit https://docs.docker.com/engine/install for more info.",
				}
				resultType = ValiationError
			} else {
				lines = []string{
					"Visit https://docs.docker.com/engine/install for more info.",
				}
				resultType = ValiationError
				category = ValidationCategoryRequired
			}

			results = append(results, ValidationResult{
				Type:     resultType,
				Category: category,
				Message:  "Docker is not installed\n" + addPrefixedSpaces(lines, 4),
			})
		} else if errors.Is(dockerErr, docker.ErrMinVersion) {
			lines := []string{
				"Visit https://docs.docker.com/engine/install to install the latest version.",
			}
			results = append(results, ValidationResult{
				Type:     ValiationError,
				Category: category,
				Message:  fmt.Sprintf("Docker is installed but minimum version must be %d.\n", docker.MinMajorVersion) + addPrefixedSpaces(lines, 4),
			})
		} else {
			results = append(results, dockerIsInstalled)
		}

		if errors.Is(dockerErr, docker.ErrDaemonNotRunning) {
			var (
				lines      []string
				resultType ValidationResultType
			)
			if runtime.GOOS == "linux" {
				lines = []string{
					"To use the `pgxman container` commands, you'll need to start the Docker daemon.",
					"Visit https://docs.docker.com/config/daemon/start for more info.",
				}
				resultType = ValidationWarning
			} else if runtime.GOOS == "darwin" {
				lines = []string{
					"pgxman emulates the production experience on macOS.",
					"To use the `pgxman install` & `pgxman container` commands, you'll need to start the Docker daemon.",
					"Visit https://docs.docker.com/config/daemon/start for more info.",
				}
				resultType = ValiationError
			} else {
				lines = []string{
					"Visit https://docs.docker.com/config/daemon/start for more info.",
				}
				resultType = ValiationError
			}

			results = append(results, ValidationResult{
				Type:     resultType,
				Category: category,
				Message:  "Docker daemon is not running\n" + addPrefixedSpaces(lines, 4),
			})
		} else {
			results = append(results, dockerIsRunning)
		}

		if len(results) == 0 {
			results = append(results, ValidationResult{
				Type:     ValiationError,
				Category: category,
				Message:  fmt.Sprintf("Docker error: %s", dockerErr.Error()),
			})
		}
	} else {
		results = append(
			results,
			dockerIsInstalled,
			dockerIsRunning,
		)
	}

	return results
}

type postgresValidator struct {
}

func (v *postgresValidator) Validate(ctx context.Context) []ValidationResult {
	var (
		results []ValidationResult
	)

	pgVer, e := pg.DetectVersion(ctx)
	if e != nil {
		lines := []string{
			"To install a PostgreSQL extension, you'll need install PostgreSQL.",
			"Visit https://docs.pgxman.com/installing_postgres for more info.",
		}
		results = append(results, ValidationResult{
			Type:     ValiationError,
			Category: ValidationCategoryRequired,
			Message:  "PostgreSQL is not installed\n" + addPrefixedSpaces(lines, 4),
		})
	} else {
		results = append(results, ValidationResult{
			Type:     ValidationSuccess,
			Category: ValidationCategoryRequired,
			Message:  fmt.Sprintf("PostgreSQL %s is installed", pgVer),
		})
	}

	return results
}

func addPrefixedSpaces(lines []string, spaces int) string {
	var result []string
	for _, line := range lines {
		result = append(result, fmt.Sprintf("%s%s", strings.Repeat(" ", spaces), line))
	}

	return strings.Join(result, "\n")
}
