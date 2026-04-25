package press

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type BuildOptions struct {
	OutDir string
	Clean  bool

	StaticDir      string
	AssetsBasePath string

	OnWrite func(path string)
}

type SiteData struct{}

type HomePageData struct {
	Site SiteData
}

func Build(opts BuildOptions, r Renderers) error {
	ctx, err := newBuildContext(opts, r)
	if err != nil {
		return err
	}

	if err := ctx.prepareOutput(); err != nil {
		return err
	}

	if err := renderSite(ctx); err != nil {
		return err
	}

	if err := ctx.syncAssets(); err != nil {
		return err
	}

	return nil
}

type buildContext struct {
	clean          bool
	outDir         string
	staticDir      string
	assetsBasePath string
	site           SiteData
	r              Renderers
	onWrite        func(path string)
}

func renderSite(ctx buildContext) error {
	if err := ctx.buildHome(); err != nil {
		return err
	}

	return nil
}

func (ctx buildContext) buildHome() error {
	if ctx.r.Home == nil {
		return fmt.Errorf("missing Home renderer")
	}

	path := filepath.Join(ctx.outDir, "index.html")
	data := HomePageData{
		Site: ctx.site,
	}

	return ctx.writeRendered(path, func(w io.Writer) error {
		return ctx.r.Home(w, data)
	})
}

func (ctx buildContext) prepareOutput() error {
	if ctx.clean {
		if err := os.RemoveAll(ctx.outDir); err != nil {
			return fmt.Errorf("clean output dir: %s: %w", ctx.outDir, err)
		}
	}

	if err := os.MkdirAll(ctx.outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %s: %w", ctx.outDir, err)
	}

	return nil
}

func (ctx buildContext) syncAssets() error {
	assetsDir := filepath.Join(ctx.outDir, ctx.assetsBasePath)
	if err := syncStaticAssets(ctx.staticDir, assetsDir, ctx.onWrite); err != nil {
		return err
	}

	return nil
}

func newBuildContext(opts BuildOptions, r Renderers) (buildContext, error) {
	outDir, err := cleanBuildPath("output dir", opts.OutDir)
	if err != nil {
		return buildContext{}, err
	}

	staticDir, err := cleanBuildPath("static dir", opts.StaticDir)
	if err != nil {
		return buildContext{}, err
	}

	assetsBasePath := opts.AssetsBasePath
	if assetsBasePath == "" {
		assetsBasePath = "assets"
	}

	assetsBasePath = filepath.Clean(assetsBasePath)
	if assetsBasePath == "." {
		return buildContext{}, fmt.Errorf("invalid assets base path: %q", opts.AssetsBasePath)
	}
	if filepath.IsAbs(assetsBasePath) {
		return buildContext{}, fmt.Errorf("assets base path must be relative: %q", opts.AssetsBasePath)
	}
	if assetsBasePath == ".." || strings.HasPrefix(assetsBasePath, ".."+string(filepath.Separator)) {
		return buildContext{}, fmt.Errorf("assets base path escapes output dir: %q", opts.AssetsBasePath)
	}

	return buildContext{
		clean:          opts.Clean,
		outDir:         outDir,
		staticDir:      staticDir,
		assetsBasePath: assetsBasePath,
		site:           SiteData{},
		r:              r,
		onWrite:        opts.OnWrite,
	}, nil
}

func cleanBuildPath(name, path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("%s is required", name)
	}

	clean := filepath.Clean(path)
	if clean == "." {
		return "", fmt.Errorf("refusing unsafe %s: %q", name, path)
	}

	return clean, nil
}
