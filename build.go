package press

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type BuildOptions struct {
	OutDir string
}

type SiteData struct{}

type HomePageData struct {
	Site SiteData
}

type Renderers struct {
	Home func(w io.Writer, data HomePageData) error
}

func Build(opts BuildOptions, r Renderers) ([]string, error) {
	out := filepath.Clean(opts.OutDir)

	if err := os.MkdirAll(out, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir: %s: %w", out, err)
	}

	site := SiteData{}

	var written []string

	if w, err := buildHome(out, site, r); err != nil {
		return nil, err
	} else {
		written = append(written, w...)
	}

	return written, nil
}

func buildHome(out string, site SiteData, r Renderers) ([]string, error) {
	if r.Home == nil {
		return nil, fmt.Errorf("missing Home renderer")
	}

	path := filepath.Join(out, "index.html")
	data := HomePageData{
		Site: site,
	}

	if err := writeRendered(path, func(w io.Writer) error {
		return r.Home(w, data)
	}); err != nil {
		return nil, err
	}

	return []string{path}, nil
}

func writeRendered(path string, render func(io.Writer) error) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir: %s: %w", filepath.Dir(path), err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create: %s: %w", path, err)
	}

	if err := render(f); err != nil {
		_ = f.Close()
		return fmt.Errorf("render: %s: %w", path, err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("close: %s: %w", path, err)
	}

	return nil
}
