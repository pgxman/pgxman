package debian

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
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

	regexpSourceFileURIs = regexp.MustCompile(`(?m)^URIs:\s*(.+)$`)
	regexpSourceListURIs = regexp.MustCompile(`\bdeb(?:-src)?\s+(?:\[.*\]\s+)?(http[^\s]+)`)
	regexpConflictDebPkg = regexp.MustCompile(`trying to overwrite '(.+)', which is also in package`)
)

type aptSourcesTmplData struct {
	Types      string
	URIs       string
	Suites     string
	Components string
	SignedBy   string
}

func NewApt(logger *log.Logger) (*Apt, error) {
	hosts, err := exitingAptSourceHosts(aptSourceDir)
	if err != nil {
		return nil, err
	}

	return &Apt{
		ExitingSourceHosts: hosts,
		Logger:             logger,
	}, nil
}

type Apt struct {
	ExitingSourceHosts map[string]struct{}
	Logger             *log.Logger
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
	Pkg       string
	IsLocal   bool
	Opts      []string
	Repos     []pgxman.AptRepository
	Overwrite bool
}

func (a *Apt) ConvertSources(ctx context.Context, repos []pgxman.AptRepository) ([]AptSource, error) {
	var result []AptSource
	for _, repo := range repos {
		uris := a.removeDuplicatedSourceHosts(repo.URIs)
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
	aptSourceMap := make(map[string]AptSource)
	for _, repo := range repos {
		uris := a.removeDuplicatedSourceHosts(repo.URIs)
		if len(uris) == 0 {
			a.Logger.Debug("Skipping apt source that already exists", "name", repo.Name())
			continue
		}
		repo.URIs = uris

		key := strings.Join(repo.URIs, ",")
		_, exists := aptSourceMap[key]
		if exists {
			a.Logger.Debug("Skipping duplicated apt source", "name", repo.Name())
			continue
		}

		file, err := a.newSourceFile(ctx, repo)
		if err != nil {
			return nil, err
		}

		diff, err := isFileDifferent(file.SourcePath, file.SourceContent)
		if err != nil {
			return nil, err
		}
		if diff {
			aptSourceMap[key] = *file
			continue
		}

		diff, err = isFileDifferent(file.KeyPath, file.KeyContent)
		if err != nil {
			return nil, err
		}
		if diff {
			aptSourceMap[key] = *file
		}
	}

	var result []AptSource
	for _, v := range aptSourceMap {
		result = append(result, v)
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

func (a *Apt) removeDuplicatedSourceHosts(uris []string) []string {
	var result []string
	for _, uri := range uris {
		u, err := sourceHost(uri)
		if err != nil {
			// impossible
			panic(err.Error())
		}

		_, exists := a.ExitingSourceHosts[u]
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
	logger := a.Logger.With("package", pkg, "upgrade", upgrade)
	logger.Debug("Running apt install or upgrade")

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

	out, oerr := a.runAptCmd(ctx, "apt", opts...)
	if oerr != nil {
		if conflictDebPkg(out) {
			if pkg.Overwrite {
				logger.Debug("Force overwriting package")
				_, oerr = a.runAptCmd(ctx, "apt", append(opts, "-o", "Dpkg::Options::=--force-overwrite")...)
			} else {
				return pgxman.ErrConflictExtension
			}
		}

		if oerr != nil {
			err = errors.Join(err, oerr)
		}
	}

	return err
}

func (a *Apt) aptUpdate(ctx context.Context) error {
	_, err := a.runAptCmd(ctx, "apt", "update")
	if err != nil {
		return fmt.Errorf("apt update: %w", err)
	}

	return nil
}

func (a *Apt) aptMarkHold(ctx context.Context, pkg AptPackage) error {
	_, err := a.runAptCmd(ctx, "apt-mark", "hold", pkg.Pkg)
	if err != nil {
		return fmt.Errorf("apt-mark hold: %w", err)
	}

	return nil
}

func (a *Apt) aptMarkUnhold(ctx context.Context, pkg AptPackage) error {
	_, err := a.runAptCmd(ctx, "apt-mark", "unhold", pkg.Pkg)
	if err != nil {
		return fmt.Errorf("apt-mark unhold: %w", err)
	}

	return nil
}

func (a *Apt) runAptCmd(ctx context.Context, command string, args ...string) (string, error) {
	c := append([]string{command}, args...)

	bw := bytes.NewBuffer(nil)
	lw := a.Logger.Writer(slog.LevelDebug)

	cmd := exec.CommandContext(ctx, c[0], c[1:]...)
	cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	cmd.Stdout = io.MultiWriter(lw, bw)
	cmd.Stderr = io.MultiWriter(lw, bw)

	a.Logger.Debug("Running apt command", "command", cmd.String())
	err := cmd.Run()

	return bw.String(), err
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

func conflictDebPkg(out string) bool {
	return regexpConflictDebPkg.MatchString(out)
}

func writeFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return os.WriteFile(path, content, 0644)
}

func coreAptRepos() ([]pgxman.AptRepository, error) {
	p, err := pgxman.DetectPlatform()
	if err != nil {
		return nil, fmt.Errorf("detect platform: %s", err)
	}

	var (
		prefix   string
		codename string
	)

	switch p {
	case pgxman.PlatformDebianBookworm:
		prefix = "debian"
		codename = "bookworm"
	case pgxman.PlatformUbuntuJammy:
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

func exitingAptSourceHosts(sourceDir string) (map[string]struct{}, error) {
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

		for _, m := range regexpSourceListURIs.FindAllStringSubmatch(string(b), -1) {
			u, err := sourceHost(m[1])
			if err != nil {
				return nil, err
			}

			result[u] = struct{}{}
		}

		for _, m := range regexpSourceFileURIs.FindAllStringSubmatch(string(b), -1) {
			for _, s := range strings.Split(m[1], " ") {
				s = strings.TrimSpace(s)
				if s == "" {
					continue
				}

				u, err := sourceHost(s)
				if err != nil {
					return nil, err
				}

				result[u] = struct{}{}
			}
		}
	}

	return result, nil
}

func sourceHost(surl string) (string, error) {
	u, err := url.Parse(path.Clean(surl))
	if err != nil {
		return "", err
	}

	// skip scheme, user info & query
	// apt ignores http:// and https:// treat them as the same host
	// for example, https://apt.postgresql.org/pub/repos/apt & http://apt.postgresql.org/pub/repos/apt are the same host
	u.Scheme = ""
	u.User = nil
	u.RawQuery = ""
	u.Fragment = ""

	return u.String(), nil
}
