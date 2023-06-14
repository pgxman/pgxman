package template

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type Templater interface {
	Render(content []byte, out io.Writer) error
}

func Export(f fs.ReadFileFS, t Templater, dstDir string) error {
	return fs.WalkDir(f, ".", func(path string, d fs.DirEntry, err error) error {
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

		// ignore embed.go
		if filepath.Base(path) == "embed.go" {
			return nil
		}

		b, err := f.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read file %w", err)
		}

		out, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("create destination file %w", err)
		}

		if t == nil {
			if _, err := io.Copy(out, bytes.NewReader(b)); err != nil {
				return fmt.Errorf("copy file %w", err)
			}
		} else {
			if err := t.Render(b, out); err != nil {
				return fmt.Errorf("render template: %w", err)
			}
		}

		if err = out.Close(); err != nil {
			return fmt.Errorf("close destination file %w", err)
		}

		return nil
	})
}
