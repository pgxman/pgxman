package pgxman

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pgxman/pgxman/internal/doctor"
	"github.com/spf13/cobra"
)

var (
	checkMark        = lipgloss.NewStyle().Foreground(lipgloss.Color("#008000")).SetString("✓")
	warningCrossMark = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00")).SetString("x")
	errorCrossMark   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).SetString("x")
)

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
		lines        = []string{"Doctor summary:"}
		failureCount int
	)

	printLine := func(result doctor.ValidationResult) string {
		var line string

		switch result.Type {
		case doctor.ValidationSuccess:
			line = fmt.Sprintf("[%s] %s", checkMark, result.Message)
		case doctor.ValidationWarning:
			line = fmt.Sprintf("[%s] %s", warningCrossMark, result.Message)
			failureCount++
		case doctor.ValiationError:
			line = fmt.Sprintf("[%s] %s", errorCrossMark, result.Message)
			failureCount++
		default:
			panic(fmt.Sprintf("unknown validation result type: %s", result.Type))
		}

		return line
	}

	required, optional := doctor.Validate(cmd.Context())
	for _, result := range required {
		lines = append(lines, printLine(result))
	}

	if len(optional) > 0 {
		lines = append(lines, "Recommendations:")

		for _, result := range optional {
			lines = append(lines, printLine(result))
		}
	}

	if failureCount == 0 {
		lines = append(lines, "\nYour system is ready to use pgxman!")
	} else {
		lines = append(lines, fmt.Sprintf("\nDoctor found %d issues.", failureCount))
	}

	fmt.Println(strings.Join(lines, "\n"))

	if failureCount > 0 {
		os.Exit(1)
	}
}
