package press

import "io"

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

func Build(opts BuildOptions) ([]string, error) {
	return []string{}, nil
}
