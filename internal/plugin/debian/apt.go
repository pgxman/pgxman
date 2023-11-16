package debian

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
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
	coreAptSourceURL       = "https://apt.pgxman.com"
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

func NewApt(sudo bool, logger *log.Logger) (*Apt, error) {
	uris, err := exitingAptSourceURIs(aptSourceDir)
	if err != nil {
		return nil, err
	}

	return &Apt{
		Sudo:              sudo,
		ExitingSourceURIs: uris,
		Logger:            logger,
	}, nil
}

type Apt struct {
	ExitingSourceURIs map[string]struct{}
	Sudo              bool
	Logger            *log.Logger
}

type AptSource struct {
	Name          string
	SourcePath    string
	SourceContent []byte
	KeyPath       string
	KeyContent    []byte
}

func (a AptSource) String() string {
	return fmt.Sprintf("%s (%s)", a.Name, a.SourcePath)
}

type AptPackage struct {
	Pkg     string
	IsLocal bool
	Opts    []string
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
	return a.installOrUpgrade(ctx, pkgs, sources, false)
}

func (a *Apt) Upgrade(ctx context.Context, pkgs []AptPackage, sources []AptSource) error {
	return a.installOrUpgrade(ctx, pkgs, sources, true)
}

func (a *Apt) installOrUpgrade(ctx context.Context, pkgs []AptPackage, sources []AptSource, upgrade bool) error {
	a.Logger.Debug("Installing or upgrading debian packages", "packages", pkgs, "sources", sources, "upgrade", upgrade)

	if err := a.addSources(ctx, sources); err != nil {
		return err
	}

	if err := a.aptInstallOrUpgrade(ctx, pkgs, upgrade); err != nil {
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

	return a.aptUpdate(ctx)
}

func (a *Apt) aptInstallOrUpgrade(ctx context.Context, pkgs []AptPackage, upgrade bool) error {
	for _, pkg := range pkgs {
		if err := a.aptInstallOrUpgradeOne(ctx, pkg, upgrade); err != nil {
			return err
		}
	}

	return nil
}

func (a *Apt) aptInstallOrUpgradeOne(ctx context.Context, pkg AptPackage, upgrade bool) (err error) {
	a.Logger.Debug("Running apt install or upgrade", "package", pkg, "upgrade", upgrade)

	// apt-mark hold and unhold don't work for a local package
	if !pkg.IsLocal {
		err = errors.Join(err, a.aptMarkUnhold(ctx, pkg))
		if err != nil {
			return err
		}

		defer func() {
			err = errors.Join(err, a.aptMarkHold(ctx, pkg))
		}()
	}

	var opts []string
	if upgrade {
		opts = []string{"upgrade", "--allow-downgrades"}
	} else {
		opts = []string{"install"}
	}

	opts = append(opts, "--yes", "--no-install-recommends")
	opts = append(opts, pkg.Opts...)
	opts = append(opts, pkg.Pkg)

	err = errors.Join(err, a.runAptCmd(ctx, "apt", opts...))

	return err
}

func (a *Apt) aptUpdate(ctx context.Context) error {
	return a.runAptCmd(ctx, "apt", "update")
}

func (a *Apt) aptMarkHold(ctx context.Context, pkg AptPackage) error {
	return a.runAptCmd(ctx, "apt-mark", "hold", pkg.Pkg)
}

func (a *Apt) aptMarkUnhold(ctx context.Context, pkg AptPackage) error {
	return a.runAptCmd(ctx, "apt-mark", "unhold", pkg.Pkg)
}

func (a *Apt) runAptCmd(ctx context.Context, command string, args ...string) error {
	var c []string
	if a.Sudo {
		c = append(c, "sudo", command)
	} else {
		c = append(c, command)
	}
	c = append(c, args...)

	w := a.Logger.Writer(slog.LevelDebug)
	cmd := exec.CommandContext(ctx, c[0], c[1:]...)
	cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	cmd.Stdout = w
	cmd.Stderr = w

	a.Logger.Debug("Running apt command", "command", cmd.String())
	return cmd.Run()
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
