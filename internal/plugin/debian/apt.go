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
	"regexp"
	"strings"
	"text/template"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/log"
)

const (
	aptKeyRingsDir = "/usr/share/keyrings"
	aptSourceDir   = "/etc/apt/sources.list.d"

	coreAptSourceGPGKeyURL = "https://pgxman.github.io/buildkit/pgxman.gpg"
	coreAptSourceURL       = "https://pgxman-buildkit-debian.s3.amazonaws.com"
)

var (
	aptSourcesTmpl = template.Must(template.New("").Parse(`Types: {{ .Types }}
URIs: {{ .URIs }}
Suites: {{ .Suites }}
Components: {{ .Components }}
Signed-By: {{ .SignedBy }}
`))

	regexpSourceURIs = regexp.MustCompile(`(?m)^URIs:\s*(.+)$`)
)

type aptSourcesTmplData struct {
	Types      string
	URIs       string
	Suites     string
	Components string
	SignedBy   string
}

func NewApt(logger *log.Logger) (*Apt, error) {
	uris, err := exitingAptSourceURIs(aptSourceDir)
	if err != nil {
		return nil, err
	}

	return &Apt{
		ExitingSourceURIs: uris,
		Logger:            logger,
	}, nil
}

type Apt struct {
	ExitingSourceURIs map[string]struct{}
	Logger            *log.Logger
}

type AptSource struct {
	Name          string
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
		uris := a.removeDuplicatedSourceURIs(repo.URIs)
		if len(uris) == 0 {
			a.Logger.Debug("Skipping apt source that already exists", "name", repo.Name())
			continue
		}
		repo.URIs = uris

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
		uris := a.removeDuplicatedSourceURIs(repo.URIs)
		if len(uris) == 0 {
			a.Logger.Debug("Skipping apt source that already exists", "name", repo.Name())
			continue
		}
		repo.URIs = uris

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

func (a *Apt) removeDuplicatedSourceURIs(uris []string) []string {
	var result []string
	for _, uri := range uris {
		_, exists := a.ExitingSourceURIs[uri]
		if !exists {
			result = append(result, uri)
		}
	}

	return result
}

func (a *Apt) addSources(ctx context.Context, files []AptSource) error {
	for _, file := range files {
		a.Logger.Debug("Writing source", "source_path", file.SourcePath, "key_path", file.KeyPath)
		if err := writeFile(file.SourcePath, file.SourceContent); err != nil {
			return err
		}
		if err := writeFile(file.KeyPath, []byte(file.KeyContent)); err != nil {
			return err
		}
	}

	return runAptUpdate(ctx)
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
	logger := a.Logger.WithGroup(repo.Name())

	keyPath := filepath.Join(aptKeyRingsDir, repo.Name()+"."+string(repo.SignedKey.Format))
	logger.Debug("Downloading gpg key", "url", repo.SignedKey, "path", keyPath)
	keyContent, err := downloadURL(repo.SignedKey.URL)
	if err != nil {
		return nil, err
	}

	sourceContent := bytes.NewBuffer(nil)
	if err := aptSourcesTmpl.Execute(sourceContent, aptSourcesTmplData{
		Types:      repo.TypesString(),
		URIs:       repo.URIsString(),
		Suites:     repo.SuitesString(),
		Components: repo.ComponentsString(),
		SignedBy:   keyPath,
	}); err != nil {
		return nil, err
	}

	return &AptSource{
		Name:          repo.Name(),
		SourcePath:    filepath.Join(aptSourceDir, repo.Name()+".sources"),
		SourceContent: sourceContent.Bytes(),
		KeyPath:       keyPath,
		KeyContent:    keyContent,
	}, nil
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

func coreAptRepos() ([]pgxman.AptRepository, error) {
	bt, err := pgxman.DetectExtensionBuilder()
	if err != nil {
		return nil, fmt.Errorf("detect platform: %s", err)
	}

	var (
		prefix   string
		codename string
	)

	switch bt {
	case pgxman.ExtensionBuilderDebianBookworm:
		prefix = "debian"
		codename = "bookworm"
	case pgxman.ExtensionBuilderUbuntuJammy:
		prefix = "ubuntu"
		codename = "jammy"
	default:
		return nil, fmt.Errorf("unsupported platform")
	}

	return []pgxman.AptRepository{
		{
			ID:         "core",
			Types:      []pgxman.AptRepositoryType{pgxman.AptRepositoryTypeDeb},
			URIs:       []string{fmt.Sprintf("%s/%s", coreAptSourceURL, prefix)},
			Suites:     []string{codename},
			Components: []string{"main"},
			SignedKey: pgxman.AptRepositorySignedKey{
				URL:    coreAptSourceGPGKeyURL,
				Format: pgxman.AptRepositorySignedKeyFormatGpg,
			},
		},
	}, nil
}

func exitingAptSourceURIs(sourceDir string) (map[string]struct{}, error) {
	files, err := os.ReadDir(sourceDir)
	if err != nil {
		return nil, err
	}

	result := make(map[string]struct{})
	for _, file := range files {
		// always ignore pgxman-core.sources so that it can be updated
		if file.Name() == "pgxman-core.sources" {
			continue
		}

		b, err := os.ReadFile(filepath.Join(sourceDir, file.Name()))
		if err != nil {
			return nil, err
		}

		for _, m := range regexpSourceURIs.FindAllStringSubmatch(string(b), -1) {
			for _, s := range strings.Split(m[1], " ") {
				s = strings.TrimSpace(s)
				if s == "" {
					continue
				}

				result[s] = struct{}{}
			}
		}
	}

	return result, nil
}
