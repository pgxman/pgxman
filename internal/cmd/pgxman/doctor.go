package pgxman

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pgxman/pgxman/internal/doctor"
	"github.com/spf13/cobra"
)

var (
	checkMark = lipgloss.NewStyle().Foreground(lipgloss.Color("#008000")).SetString("✓")
	crossMark = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).SetString("x")
)

const (
	ValidationSuccess ValidationResult = "success"
	ValiationError    ValidationResult = "error"
)

type ValidationResult string

type ValidateFunc func() (ValidationResult, string)

func newDoctorCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "doctor",
		Short: "Troubleshoot your system for potential problems",
		Long:  `Check your system for potential problems. Will exit with a non-zero status if any potential problems are found.`,
		Run:   runDoctor,
	}

	return root
}

func runDoctor(cmd *cobra.Command, args []string) {
	var (
		lines        []string
		failureCount int
	)

	results := doctor.Validate(cmd.Context())
	for _, result := range results {
		var line string

		switch result.Type {
		case doctor.ValidationSuccess:
			line = fmt.Sprintf("[%s] %s", checkMark, result.Message)
		case doctor.ValiationError:
			line = fmt.Sprintf("[%s] %s", crossMark, result.Message)
			failureCount++
		default:
			panic(fmt.Sprintf("unknown validation result type: %s", result.Type))
		}

		lines = append(lines, line)
	}

	lines = append([]string{"Doctor summary:"}, lines...)
	if failureCount == 0 {
		lines = append(lines, "\nYour system is ready to use pgxman!")
	} else {
		lines = append(lines, fmt.Sprintf("\nDoctor found %d issues.", failureCount))
	}

	fmt.Println(strings.Join(lines, "\n"))
}
