package press

import (
	"fmt"
	"net/http"
	"os"
)

type ServeOptions struct {
	ServeDir string
	Addr     string
}

func Serve(opts ServeOptions) error {
	dir := opts.ServeDir
	addr := opts.Addr

	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("stat: %s: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", dir)
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(dir)))

	if err := http.ListenAndServe(addr, mux); err != nil {
		return fmt.Errorf("listen: %s: %w", addr, err)
	}

	return nil
}
