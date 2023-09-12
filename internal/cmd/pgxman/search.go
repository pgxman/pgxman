package pgxman

import (
	"fmt"
	"os"
	"regexp"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pgxman/pgxman/internal/buildkit"
	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "search [search terms ...]",
		Aliases: []string{"s"},
		Short:   "Search for extensions",
		Long:    `Search for installable PostgreSQL extensions.`,
		Example: `  # Search for pgvector
  pgxman search pgvector
		`,
		Args: cobra.MinimumNArgs(1),
		RunE: runSearch,
	}

	return cmd
}

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type searchModel struct {
	table table.Model
}

func (m searchModel) Init() tea.Cmd { return nil }

func (m searchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			return m, tea.Batch(
				tea.Printf("Let's go to %s!", m.table.SelectedRow()[1]),
			)
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m searchModel) View() string {
	return baseStyle.Render(m.table.View()) + "\n"
}

func runSearch(c *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no search terms provided")
	}

	re, err := regexp.Compile(args[0])
	if err != nil {
		return err
	}

	exts, err := buildkit.InstallableExtensions(c.Context())
	if err != nil {
		return err
	}

	var rows []table.Row
	for _, ext := range exts {
		if re.MatchString(ext.Name) || re.MatchString(ext.Description) {
			rows = append(rows, table.Row{
				ext.Name,
				ext.Version,
				ext.Description,
			})
		}
	}

	if len(rows) == 0 {
		fmt.Println("No extensions found.")
		return nil
	}

	columns := []table.Column{
		{Title: "Name", Width: 10},
		{Title: "Version", Width: 10},
		{Title: "Description", Width: 20},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(7),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	m := searchModel{t}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	return nil
}
