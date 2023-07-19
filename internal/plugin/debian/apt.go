package debian

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/log"
	"github.com/pgxman/pgxman/internal/osx"
)

var (
	keyringsDir    = "/usr/share/keyrings"
	sourceListdDir = "/etc/apt/sources.list.d"
	aptSourcesTmpl = template.Must(template.New("").Parse(`Types: {{ .Types }}
URIs: {{ .URIs }}
Suites: {{ .Suites }}
Components: {{ .Components }}
Signed-By: {{ .SignedBy }}
`))
)

type aptSourcesTmplData struct {
	Types      string
	URIs       string
	Suites     string
	Components string
	SignedBy   string
}

func addAptRepos(ctx context.Context, repos []pgxman.AptRepository, logger *log.Logger) error {
	for _, repo := range repos {
		logger := logger.WithGroup(repo.ID)
		logger.Debug("Adding apt repo")

		ext := filepath.Ext(repo.SignedKey)
		if ext == "" {
			ext = ".asc"
		}

		gpgKeyPath := filepath.Join(keyringsDir, repo.ID+ext)
		logger.Debug("Downloading gpg key", "url", repo.SignedKey, "path", gpgKeyPath)
		if err := downloadFile(repo.SignedKey, gpgKeyPath); err != nil {
			return err
		}

		var (
			suites []string
			osn    = osx.Sysinfo().OS.Name
		)
		for _, suite := range repo.Suites {
			// if no target is specified, the suite is valid for all OSs
			// otherwise, the suite is only valid for the specified OS
			if suite.Target == "" || strings.Contains(osn, suite.Target) {
				suites = append(suites, suite.Suite)
			}
		}

		b := bytes.NewBuffer(nil)
		if err := aptSourcesTmpl.Execute(b, aptSourcesTmplData{
			Types:      strings.Join(repo.Types, " "),
			URIs:       strings.Join(repo.URIs, " "),
			Suites:     strings.Join(suites, " "),
			Components: strings.Join(repo.Components, " "),
			SignedBy:   gpgKeyPath,
		}); err != nil {
			return err
		}

		sourcesPath := filepath.Join(sourceListdDir, repo.ID+".sources")
		logger.Debug("Writing source", "path", sourcesPath, "content", b.String())
		if err := writeFile(sourcesPath, b.Bytes()); err != nil {
			return err
		}
	}

	return runAptUpdate(ctx)
}

func downloadFile(url, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return writeFile(path, b)
}

func writeFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return os.WriteFile(path, content, 0644)
}

func runAptUpdate(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "apt", "update")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
