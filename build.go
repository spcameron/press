package press

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
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

type Renderers struct {
	Home func(w io.Writer, data HomePageData) error
}

type buildContext struct {
	out            string
	staticDir      string
	assetsBasePath string
	site           SiteData
	r              Renderers
	onWrite        func(path string)
}

func Build(opts BuildOptions, r Renderers) error {
	ctx, err := newBuildContext(opts, r)
	if err != nil {
		return err
	}

	if opts.Clean {
		if err := os.RemoveAll(ctx.out); err != nil {
			return fmt.Errorf("clean output dir %s: %w", ctx.out, err)
		}
	}

	if err := os.MkdirAll(ctx.out, 0o755); err != nil {
		return fmt.Errorf("mkdir: %s: %w", ctx.out, err)
	}

	if err := renderSite(ctx); err != nil {
		return err
	}

	assetsDir := filepath.Join(ctx.out, ctx.assetsBasePath)
	if err := syncStaticAssets(ctx.staticDir, assetsDir, ctx.onWrite); err != nil {
		return err
	}

	return nil
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

	path := filepath.Join(ctx.out, "index.html")
	data := HomePageData{
		Site: ctx.site,
	}

	return ctx.writeRendered(path, func(w io.Writer) error {
		return ctx.r.Home(w, data)
	})
}

func (ctx buildContext) writeRendered(path string, render func(io.Writer) error) error {
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

	if ctx.onWrite != nil {
		ctx.onWrite(path)
	}

	return nil
}

func syncStaticAssets(srcDir, dstDir string, onWrite func(path string)) error {
	srcInfo, err := os.Stat(srcDir)
	if err != nil {
		return fmt.Errorf("stat static dir %s: %w", srcDir, err)
	}
	if !srcInfo.IsDir() {
		return fmt.Errorf("static path is not a directory: %s", srcDir)
	}

	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return fmt.Errorf("create assets dir %s: %w", dstDir, err)
	}

	want := make(map[string]struct{})

	if err := filepath.WalkDir(srcDir, func(srcPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk static dir: %w", walkErr)
		}

		rel, err := filepath.Rel(srcDir, srcPath)
		if err != nil {
			return fmt.Errorf("rel static path: %w", err)
		}
		if rel == "." {
			return nil
		}

		want[rel] = struct{}{}

		dstPath := filepath.Join(dstDir, rel)

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("stat static entry %s: %w", srcPath, err)
		}

		switch {
		case info.Mode()&os.ModeSymlink != 0:
			return fmt.Errorf("refusing to copy symlink: %s", srcPath)

		case d.IsDir():
			if err := ensureDirPath(dstPath, info.Mode().Perm()); err != nil {
				return fmt.Errorf("create asset dir %s: %w", dstPath, err)
			}
			return nil

		case info.Mode().IsRegular():
			if err := ensureFilePath(dstPath); err != nil {
				return fmt.Errorf("prepare asset file %s: %w", dstPath, err)
			}
			if err := copyFile(srcPath, dstPath, info.Mode().Perm()); err != nil {
				return fmt.Errorf("copy asset %s: %w", srcPath, err)
			}
			if onWrite != nil {
				onWrite(dstPath)
			}
			return nil

		default:
			return fmt.Errorf("refusing to copy non-regular static entry: %s", srcPath)
		}
	}); err != nil {
		return err
	}

	var stale []string

	if err := filepath.WalkDir(dstDir, func(dstPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk assets dir: %w", walkErr)
		}

		rel, err := filepath.Rel(dstDir, dstPath)
		if err != nil {
			return fmt.Errorf("rel assets path: %w", err)
		}
		if rel == "." {
			return nil
		}

		if _, ok := want[rel]; !ok {
			stale = append(stale, dstPath)
		}

		return nil
	}); err != nil {
		return err
	}

	sort.Slice(stale, func(i, j int) bool {
		return len(stale[i]) > len(stale[j])
	})

	for _, path := range stale {
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("remove stale asset %s: %w", path, err)
		}
	}

	return nil
}

func ensureDirPath(path string, mode fs.FileMode) error {
	info, err := os.Lstat(path)
	if err == nil {
		if info.IsDir() {
			return os.Chmod(path, mode)
		}
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	return os.MkdirAll(path, mode)
}

func ensureFilePath(path string) error {
	info, err := os.Lstat(path)
	if err == nil {
		if info.IsDir() {
			if err := os.RemoveAll(path); err != nil {
				return err
			}
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	return os.MkdirAll(filepath.Dir(path), 0o755)
}

func copyFile(srcPath, dstPath string, mode fs.FileMode) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()

	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}

	return nil
}

func newBuildContext(opts BuildOptions, r Renderers) (buildContext, error) {
	out, err := cleanBuildPath("output dir", opts.OutDir)
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
		out:            out,
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
