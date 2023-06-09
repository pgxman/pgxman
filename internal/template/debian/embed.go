package debian

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/hydradatabase/pgxm/internal/template"
)

//go:embed all:*
var debianFS embed.FS

func Export(t template.Template, dstDir string) error {
	return fs.WalkDir(debianFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fs.SkipDir
		}

		dst := filepath.Join(dstDir, path)

		if d.IsDir() {
			if err = os.MkdirAll(dst, 0755); err != nil {
				return fmt.Errorf("cannot mkdir %w", err)
			}

			return nil
		}

		if filepath.Base(path) == "embed.go" {
			return nil
		}

		b, err := debianFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("cannot read file %w", err)
		}

		out, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("cannot create dst file %w", err)
		}

		if err := t.Apply(b, out); err != nil {
			return fmt.Errorf("cannot apply template: %w", err)
		}

		if err = out.Close(); err != nil {
			return fmt.Errorf("cannot close dst file %w", err)
		}

		return nil
	})
}
