package debian

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/log"
)

const (
	keyringsDir    = "/usr/share/keyrings"
	sourceListdDir = "/etc/apt/sources.list.d"
)

var (
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

type Apt struct {
	Logger *log.Logger
}

type AptSource struct {
	ID            string
	SourcePath    string
	SourceContent []byte
	KeyPath       string
	KeyContent    []byte
}

type AptPackage struct {
	Pkg  string
	Opts []string
}

func (a *Apt) ConvertSources(ctx context.Context, repos []pgxman.AptRepository) ([]AptSource, error) {
	var result []AptSource
	for _, repo := range repos {
		file, err := a.newSourceFile(ctx, repo)
		if err != nil {
			return nil, err
		}

		result = append(result, *file)
	}

	return result, nil
}

func (a *Apt) GetChangedSources(ctx context.Context, repos []pgxman.AptRepository) ([]AptSource, error) {
	var result []AptSource
	for _, repo := range repos {
		file, err := a.newSourceFile(ctx, repo)
		if err != nil {
			return nil, err
		}

		diff, err := isFileDifferent(file.SourcePath, file.SourceContent)
		if err != nil {
			return nil, err
		}
		if diff {
			result = append(result, *file)
			continue
		}

		diff, err = isFileDifferent(file.KeyPath, file.KeyContent)
		if err != nil {
			return nil, err
		}
		if diff {
			result = append(result, *file)
		}
	}

	return result, nil
}

func (a *Apt) Install(ctx context.Context, pkgs []AptPackage, sources []AptSource) error {
	a.Logger.Debug("Installing debian packages", "packages", pkgs, "sources", sources)
	if err := a.addSources(ctx, sources); err != nil {
		return err
	}
	if err := a.install(ctx, pkgs); err != nil {
		return err
	}

	return nil
}

func (a *Apt) addSources(ctx context.Context, files []AptSource) error {
	for _, file := range files {
		a.Logger.Debug("Writing source", "source_path", file.SourcePath, "key_path", file.KeyPath)
		if err := writeFile(file.SourcePath, file.SourceContent); err != nil {
			return err
		}
		if err := writeFile(file.KeyPath, []byte(file.KeyPath)); err != nil {
			return err
		}
	}

	return nil
}

func (a *Apt) install(ctx context.Context, pkgs []AptPackage) error {
	for _, pkg := range pkgs {
		a.Logger.Debug("Running apt install", "package", pkg)

		opts := []string{"install", "-y", "--no-install-recommends"}
		opts = append(opts, pkg.Opts...)
		opts = append(opts, pkg.Pkg)

		cmd := exec.CommandContext(ctx, "apt", opts...)
		cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("apt install: %w", err)
		}
	}

	return nil
}

func (a *Apt) newSourceFile(ctx context.Context, repo pgxman.AptRepository) (*AptSource, error) {
	logger := a.Logger.WithGroup(repo.ID)

	keyPath := filepath.Join(keyringsDir, repo.ID+"."+string(repo.SignedKey.Format))
	logger.Debug("Downloading gpg key", "url", repo.SignedKey, "path", keyPath)
	keyContent, err := downloadURL(repo.SignedKey.URL)
	if err != nil {
		return nil, err
	}

	var types []string
	for _, t := range repo.Types {
		types = append(types, string(t))
	}

	sourceContent := bytes.NewBuffer(nil)
	if err := aptSourcesTmpl.Execute(sourceContent, aptSourcesTmplData{
		Types:      strings.Join(types, " "),
		URIs:       strings.Join(repo.URIs, " "),
		Suites:     strings.Join(repo.Suites, " "),
		Components: strings.Join(repo.Components, " "),
		SignedBy:   keyPath,
	}); err != nil {
		return nil, err
	}

	return &AptSource{
		ID:            repo.ID,
		SourcePath:    filepath.Join(sourceListdDir, repo.ID+".sources"),
		SourceContent: sourceContent.Bytes(),
		KeyPath:       keyPath,
		KeyContent:    keyContent,
	}, nil
}

func addAptRepos(ctx context.Context, repos []pgxman.AptRepository, logger *log.Logger) error {
	for _, repo := range repos {
		logger := logger.WithGroup(repo.ID)
		logger.Debug("Adding apt repo")

		gpgKeyPath := filepath.Join(keyringsDir, repo.ID+"."+string(repo.SignedKey.Format))
		logger.Debug("Downloading gpg key", "url", repo.SignedKey, "path", gpgKeyPath)
		if err := downloadFile(repo.SignedKey.URL, gpgKeyPath); err != nil {
			return err
		}

		var types []string
		for _, t := range repo.Types {
			types = append(types, string(t))
		}

		b := bytes.NewBuffer(nil)
		if err := aptSourcesTmpl.Execute(b, aptSourcesTmplData{
			Types:      strings.Join(types, " "),
			URIs:       strings.Join(repo.URIs, " "),
			Suites:     strings.Join(repo.Suites, " "),
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

func downloadURL(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func isFileDifferent(path string, content []byte) (bool, error) {
	c, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return true, nil
		}

		return false, err
	}

	if !bytes.Equal(c, content) {
		return true, nil
	}

	return false, nil

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
