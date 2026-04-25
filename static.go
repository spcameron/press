package press

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

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
			if err := ensureDir(dstPath, info.Mode().Perm()); err != nil {
				return fmt.Errorf("create asset dir %s: %w", dstPath, err)
			}
			return nil

		case info.Mode().IsRegular():
			if err := ensureFile(dstPath); err != nil {
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

func syncDirIfExists(srcDir, dstDir string, onWrite func(path string)) error {
	info, err := os.Stat(srcDir)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.RemoveAll(dstDir); err != nil {
				return fmt.Errorf("remove stale dir %s: %w", dstDir, err)
			}
			return nil
		}
		return fmt.Errorf("stat dir %s: %w", srcDir, err)
	}
	if !info.IsDir() {
		if err := os.RemoveAll(dstDir); err != nil {
			return fmt.Errorf("remove stale dir %s: %w", dstDir, err)
		}
		return nil
	}

	want := make(map[string]struct{})

	if walkErr := filepath.WalkDir(srcDir, func(srcPath string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %s: %w", srcPath, err)
		}

		rel, err := filepath.Rel(srcDir, srcPath)
		if err != nil {
			return fmt.Errorf("rel dir path: %w", err)
		}
		if rel == "." {
			return nil
		}

		want[rel] = struct{}{}

		dstPath := filepath.Join(dstDir, rel)

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("stat entry %s: %w", srcPath, err)
		}

		switch {
		case info.Mode()&os.ModeSymlink != 0:
			return fmt.Errorf("refusing to copy symlink: %s", srcPath)

		case d.IsDir():
			if err := ensureDir(dstPath, info.Mode().Perm()); err != nil {
				return fmt.Errorf("create dir %s: %w", dstPath, err)
			}
			return nil

		case info.Mode().IsRegular():
			if err := ensureFile(dstPath); err != nil {
				return fmt.Errorf("prepare file %s: %w", dstPath, err)
			}
			if err := copyFile(srcPath, dstPath, info.Mode().Perm()); err != nil {
				return fmt.Errorf("copy file %s: %w", srcPath, err)
			}
			if onWrite != nil {
				onWrite(dstPath)
			}
			return nil

		default:
			return fmt.Errorf("refusing to copy non-regular entry: %s", srcPath)
		}
	}); walkErr != nil {
		return walkErr
	}

	var stale []string

	if walkErr := filepath.WalkDir(dstDir, func(dstPath string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk dst dir: %w", err)
		}

		rel, err := filepath.Rel(dstDir, dstPath)
		if err != nil {
			return fmt.Errorf("rel dst path: %w", err)
		}
		if rel == "." {
			return nil
		}

		if _, ok := want[rel]; !ok {
			stale = append(stale, dstPath)
		}

		return nil
	}); walkErr != nil {
		if os.IsNotExist(walkErr) {
			return nil
		}
		return walkErr
	}

	sort.Slice(stale, func(i, j int) bool {
		return len(stale[i]) > len(stale[j])
	})

	for _, path := range stale {
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("remove stale path %s: %w", path, err)
		}
	}

	return nil
}

func ensureDir(path string, mode fs.FileMode) error {
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

func ensureFile(path string) error {
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
