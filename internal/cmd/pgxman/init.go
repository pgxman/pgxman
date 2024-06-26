package pgxman

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pgxman/pgxman"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a buildkit file",
		Long: `Create a buildkit YAML file that specifies how a PostgreSQL extension is built and packaged. The
specification is available at https://docs.pgxman.com/spec/buildkit.`,
		RunE: runInit,
	}

	return cmd
}

func runInit(c *cobra.Command, args []string) error {
	user, err := user.Current()
	if err != nil {
		return err
	}

	ext := &pgxman.Extension{
		APIVersion: pgxman.DefaultExtensionAPIVersion,
		ExtensionCommon: pgxman.ExtensionCommon{
			Name:        "my-pg-extension",
			Description: "My PostgreSQL extension",
			License:     "PostgreSQL",
			Maintainers: []pgxman.Maintainer{
				{
					Name:  user.Name,
					Email: user.Username + "@localhost",
				},
			},
		},
		ExtensionOverridable: pgxman.ExtensionOverridable{
			Build: pgxman.Build{
				Main: []pgxman.BuildScript{
					{
						Name: "my build step",
						Run: `# Uncomment to write the build script for the extension.
# The built extension must be installed in the $DESTDIR directory.
# See https://docs.pgxman.com/spec/buildkit#build for details.
`,
					},
				},
			},
			Version: "1.0.0",
		},
		PGVersions: pgxman.SupportedPGVersions,
	}

	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	p := tea.NewProgram(initialModel(pwd, ext))
	_, err = p.Run()

	return err
}

var (
	focusedStyle = lipgloss.NewStyle()
	blurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	helpStyle    = blurredStyle

	focusedButton = focusedStyle.Render("> [ Done ]")
	blurredButton = blurredStyle.Render("[ Done ]")
)

type initInput struct {
	UpdateExt func(ext *pgxman.Extension, val string)

	Label string
	textinput.Model
}

type initModel struct {
	err        error
	ext        *pgxman.Extension
	extPath    string
	inputs     []initInput
	focusIndex int
	done       bool
}

func initialModel(pwd string, ext *pgxman.Extension) initModel {
	m := initModel{
		ext:        ext,
		extPath:    filepath.Join(pwd, "extension.yaml"),
		focusIndex: 0,
		inputs:     make([]initInput, 6),
	}

	for i := range m.inputs {
		t := initInput{Model: textinput.New()}
		t.PromptStyle = blurredStyle
		t.TextStyle = blurredStyle

		switch i {
		case 0:
			t.Label = "Name"
			t.Placeholder = ext.Name
			t.UpdateExt = func(ext *pgxman.Extension, val string) {
				ext.Name = val
			}

			t.TextStyle = focusedStyle
			t.PromptStyle = focusedStyle
			t.Focus()
		case 1:
			t.Label = "Description"
			t.Placeholder = ext.Description
			t.UpdateExt = func(ext *pgxman.Extension, val string) {
				ext.Description = val
			}
		case 2:
			t.Label = "Version"
			t.Placeholder = ext.Version
			t.UpdateExt = func(ext *pgxman.Extension, val string) {
				ext.Version = val
			}
		case 3:
			t.Label = "License"
			t.Placeholder = ext.License
			t.UpdateExt = func(ext *pgxman.Extension, val string) {
				ext.License = val
			}
		case 4:
			t.Label = "Keywords (comma-separated)"
			t.Placeholder = strings.Join(ext.Keywords, ",")
			t.UpdateExt = func(ext *pgxman.Extension, val string) {
				ext.Keywords = splitString(val)
			}
		case 5:
			t.Label = "Source URL"
			t.Placeholder = ext.Source
			t.UpdateExt = func(ext *pgxman.Extension, val string) {
				ext.Source = val
			}
		case 6:
			t.Label = "PG versions (comma-separated)"

			var pgvs []string
			for _, pgv := range ext.PGVersions {
				pgvs = append(pgvs, string(pgv))
			}
			t.Placeholder = strings.Join(pgvs, ",")
			t.UpdateExt = func(ext *pgxman.Extension, val string) {
				var pgvs []pgxman.PGVersion
				for _, pgv := range splitString(val) {
					pgvs = append(pgvs, pgxman.PGVersion(pgv))
				}

				ext.PGVersions = pgvs
			}
		}

		if i == 0 {
			t.Prompt = focusedPrompt(t.Label)
		} else {
			t.Prompt = blurredPrompt(t.Label)
		}

		m.inputs[i] = t
	}

	return m
}

func (m initModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m initModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			// Did the user press enter while the Done button was focused?
			// If so, exit.
			if s == "enter" && m.focusIndex == len(m.inputs) {
				m.done = true

				if err := pgxman.WriteExtension(m.extPath, *m.ext); err != nil {
					m.err = err
				}

				return m, tea.Quit
			}

			// Cycle indexes
			if s == "shift+tab" || s == "up" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > len(m.inputs) {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs)
			}

			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i <= len(m.inputs)-1; i++ {
				if i == m.focusIndex {
					// Set focused state
					cmds[i] = m.inputs[i].Focus()
					m.inputs[i].Prompt = focusedPrompt(m.inputs[i].Label)
					m.inputs[i].TextStyle = focusedStyle
					m.inputs[i].PromptStyle = focusedStyle
					continue
				}

				// Remove focused state
				m.inputs[i].Blur()
				m.inputs[i].Prompt = blurredPrompt(m.inputs[i].Label)
				m.inputs[i].TextStyle = blurredStyle
				m.inputs[i].PromptStyle = blurredStyle
			}

			return m, tea.Batch(cmds...)
		}
	}

	// Handle character input and blinking
	cmd := m.updateInputs(msg)

	return m, cmd
}

func (m *initModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))

	// Only text inputs with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	for i := range m.inputs {
		m.inputs[i].Model, cmds[i] = m.inputs[i].Update(msg)
		val := m.inputs[i].Value()
		if val != "" {
			m.inputs[i].UpdateExt(m.ext, val)
		}
	}

	return tea.Batch(cmds...)
}

func (m initModel) View() string {
	var b strings.Builder

	if m.err != nil {
		b.WriteString("Error: " + m.err.Error() + "\n")
		return b.String()
	}

	if m.done {
		b.WriteString(fmt.Sprintf("Generated %s.\n", filepath.Base(m.extPath)))
		return b.String()
	}
	b.WriteString(`This utility will walk you through creating a buildkit file.
It only covers the most common items and tries to guess sensible defaults.
See https://docs.pgxman.com/spec/buildkit for documentation.

`)

	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		if i < len(m.inputs)-1 {
			b.WriteRune('\n')
		}
	}

	button := &blurredButton
	if m.focusIndex == len(m.inputs) {
		button = &focusedButton
	}
	fmt.Fprintf(&b, "\n\n%s\n\n", *button)

	b.WriteString(helpStyle.Render("(ctrl+c to quit, enter to submit, up/down to navigate)"))

	return b.String()
}

func splitString(s string) []string {
	var result []string
	for _, s := range strings.Split(s, ",") {
		result = append(result, strings.TrimSpace(s))
	}

	return result
}

func focusedPrompt(s string) string {
	return fmt.Sprintf("> %s: ", s)
}

func blurredPrompt(s string) string {
	return fmt.Sprintf("%s: ", s)
}
