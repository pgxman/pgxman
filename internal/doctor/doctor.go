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

func Validate(ctx context.Context) []ValidationResult {
	var (
		results    []ValidationResult
		validators = []Validator{&dockerValidator{}}
	)
	if runtime.GOOS == "linux" {
		validators = append(validators, &postgresValidator{})
	}

	for _, v := range validators {
		results = append(results, v.Validate(ctx)...)
	}

	return results
}

const (
	ValidationSuccess ValidationType = "success"
	ValiationError    ValidationType = "error"
)

type ValidationType string

type ValidationResult struct {
	Type    ValidationType
	Message string
}

type Validator interface {
	Validate(context.Context) []ValidationResult
}

type dockerValidator struct {
}

func (v *dockerValidator) Validate(ctx context.Context) []ValidationResult {
	var (
		results           []ValidationResult
		dockerIsInstalled = ValidationResult{
			Type:    ValidationSuccess,
			Message: "Docker is installed",
		}
		dockerIsRunning = ValidationResult{
			Type:    ValidationSuccess,
			Message: `Docker deamon is running`,
		}
	)

	dockerErr := docker.CheckInstall(ctx)
	if dockerErr != nil {
		if errors.Is(dockerErr, docker.ErrDockerNotRunning) || errors.Is(dockerErr, docker.ErrDockerNotFound) {
			if errors.Is(dockerErr, docker.ErrDockerNotFound) {
				var lines []string
				if runtime.GOOS == "linux" {
					lines = []string{
						"To use the `pgxman container` commands, you'll need to install Docker.",
						"Visit https://docs.docker.com/engine/install for more info.",
					}
				} else if runtime.GOOS == "darwin" {
					lines = []string{
						"pgxman emulates the production experience on macOS.",
						"To use the `pgxman install` & `pgxman container` commands, you'll need to install Docker.",
						"Visit https://docs.docker.com/engine/install for more info.",
					}
				} else {
					lines = []string{
						"Visit https://docs.docker.com/engine/install for more info.",
					}
				}

				results = append(results, ValidationResult{
					Type:    ValiationError,
					Message: "Docker is not installed\n" + addPrefixSpaces(lines, 4),
				})
			} else {
				results = append(results, dockerIsInstalled)
			}

			if errors.Is(dockerErr, docker.ErrDockerNotRunning) {
				var lines []string
				if runtime.GOOS == "linux" {
					lines = []string{
						"To use the `pgxman container` commands, you'll need to start the Docker daemon.",
						"Visit https://docs.docker.com/config/daemon/start for more info.",
					}
				} else if runtime.GOOS == "darwin" {
					lines = []string{
						"pgxman emulates the production experience on macOS.",
						"To use the `pgxman install` & `pgxman container` commands, you'll need to start the Docker daemon.",
						"Visit https://docs.docker.com/config/daemon/start for more info.",
					}
				} else {
					lines = []string{
						"Visit https://docs.docker.com/config/daemon/start for more info.",
					}
				}

				results = append(results, ValidationResult{
					Type:    ValiationError,
					Message: "Docker daemon is not running\n" + addPrefixSpaces(lines, 4),
				})
			} else {
				results = append(results, dockerIsRunning)
			}
		} else {
			results = append(results, ValidationResult{
				Type:    ValiationError,
				Message: fmt.Sprintf("Docker error: %s", dockerErr.Error()),
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
			"Visit https://www.postgresql.org/download for more info.",
		}
		results = append(results, ValidationResult{
			Type:    ValiationError,
			Message: "PostgreSQL is not installed\n" + addPrefixSpaces(lines, 4),
		})
	} else {
		results = append(results, ValidationResult{
			Type:    ValidationSuccess,
			Message: fmt.Sprintf("PostgreSQL %s is installed", pgVer),
		})
	}

	return results
}

func addPrefixSpaces(lines []string, spaces int) string {
	var result []string
	for _, line := range lines {
		result = append(result, fmt.Sprintf("%s%s", strings.Repeat(" ", spaces), line))
	}

	return strings.Join(result, "\n")
}
