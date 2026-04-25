package press

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

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
