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

	ContentDir     string
	StaticDir      string
	AssetsBasePath string

	OnWrite func(path string)
}

type SiteData struct {
	Title         string
	StylesheetURL string
}

type PageData struct {
	Site  SiteData
	Title string
}

type HomePageData struct {
	Page PageData
}

type BlogIndexPageData struct {
	Page  PageData
	Posts []Post
}

type BlogPostPageData struct {
	Page PageData
	Post Post
}

func Build(opts BuildOptions, site SiteData, r Renderers) error {
	ctx, err := newBuildContext(opts, site, r)
	if err != nil {
		return err
	}

	if err := ctx.prepareOutput(); err != nil {
		return err
	}

	if err := ctx.loadPosts(); err != nil {
		return err
	}

	if err := ctx.renderSite(); err != nil {
		return err
	}

	if err := ctx.syncAssets(); err != nil {
		return err
	}

	return nil
}

type buildContext struct {
	outDir         string
	clean          bool
	contentDir     string
	staticDir      string
	assetsBasePath string

	site  SiteData
	posts []Post

	r       Renderers
	onWrite func(path string)
}

func (ctx *buildContext) renderSite() error {
	if err := ctx.buildHome(); err != nil {
		return err
	}

	if err := ctx.buildBlogIndex(); err != nil {
		return err
	}

	if err := ctx.buildBlogPosts(); err != nil {
		return err
	}

	return nil
}

func (ctx *buildContext) loadPosts() error {
	candidates, err := discoverPosts(ctx.contentDir)
	if err != nil {
		return err
	}

	posts, err := parsePosts(candidates)
	if err != nil {
		return err
	}

	if err := validatePosts(posts); err != nil {
		return err
	}

	sortPosts(posts)
	assignPostURLs(posts)

	ctx.posts = posts
	return nil
}

func (ctx *buildContext) buildHome() error {
	if ctx.r.Home == nil {
		return fmt.Errorf("missing Home renderer")
	}

	path := filepath.Join(ctx.outDir, "index.html")
	data := HomePageData{
		Page: PageData{
			Site:  ctx.site,
			Title: "Home",
		},
	}

	return ctx.writeRendered(path, func(w io.Writer) error {
		return ctx.r.Home(w, data)
	})
}

func (ctx *buildContext) buildBlogIndex() error {
	if ctx.r.BlogIndex == nil {
		return fmt.Errorf("missing BlogIndex renderer")
	}

	path := blogIndexOutputPath(ctx.outDir)
	data := BlogIndexPageData{
		Page: PageData{
			Site:  ctx.site,
			Title: "Blog",
		},
		Posts: ctx.posts,
	}

	return ctx.writeRendered(path, func(w io.Writer) error {
		return ctx.r.BlogIndex(w, data)
	})
}

func (ctx *buildContext) buildBlogPosts() error {
	if ctx.r.BlogPost == nil {
		return fmt.Errorf("missing BlogPost renderer")
	}

	for _, p := range ctx.posts {
		path := postOutputPath(ctx.outDir, p.Slug)

		data := BlogPostPageData{
			Page: PageData{
				Site:  ctx.site,
				Title: p.Title,
			},
			Post: p,
		}

		if err := ctx.writeRendered(path, func(w io.Writer) error {
			return ctx.r.BlogPost(w, data)
		}); err != nil {
			return err
		}

		if err := ctx.syncPostMedia(p); err != nil {
			return err
		}
	}

	return nil
}

func (ctx *buildContext) prepareOutput() error {
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

func (ctx *buildContext) syncAssets() error {
	assetsDir := filepath.Join(ctx.outDir, ctx.assetsBasePath)
	if err := syncStaticAssets(ctx.staticDir, assetsDir, ctx.onWrite); err != nil {
		return err
	}

	return nil
}

func (ctx *buildContext) syncPostMedia(p Post) error {
	if err := syncDirIfExists(
		postMediaSourceDir(p),
		postMediaOutputDir(ctx.outDir, p.Slug),
		ctx.onWrite,
	); err != nil {
		return err
	}

	return nil
}

func newBuildContext(opts BuildOptions, site SiteData, r Renderers) (buildContext, error) {
	outDir, err := cleanBuildPath("output dir", opts.OutDir)
	if err != nil {
		return buildContext{}, err
	}

	staticDir, err := cleanBuildPath("static dir", opts.StaticDir)
	if err != nil {
		return buildContext{}, err
	}

	contentDir, err := cleanBuildPath("content dir", opts.ContentDir)
	if err != nil {
		return buildContext{}, err
	}

	assetsBasePath, err := cleanRelativeBuildPath("assets base path", opts.AssetsBasePath)
	if err != nil {
		return buildContext{}, err
	}

	return buildContext{
		clean:          opts.Clean,
		outDir:         outDir,
		contentDir:     contentDir,
		staticDir:      staticDir,
		assetsBasePath: assetsBasePath,
		site:           site,
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

func cleanRelativeBuildPath(name, path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("%s is required", name)
	}

	clean := filepath.Clean(path)
	if clean == "." {
		return "", fmt.Errorf("refusing unsafe %s: %q", name, path)
	}
	if filepath.IsAbs(clean) {
		return "", fmt.Errorf("%s must be relative: %q", name, path)
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%s escapes output dir: %q", name, path)
	}

	return clean, nil
}
