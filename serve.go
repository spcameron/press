package press

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

type ServeOptions struct {
	Dir  string
	Addr string
}

func Serve(opts ServeOptions) error {
	dir := opts.Dir
	addr := opts.Addr

	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("stat: %s: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", dir)
	}

	mux := http.NewServeMux()
	fileServer := http.FileServer(neuteredFileSystem{fs: http.Dir(dir)})
	mux.Handle("/", fileServer)

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	fmt.Printf("serve: serving %s on http://%s\n", dir, addr)

	errCh := make(chan error, 1)
	go func() {
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("listen: %s: %w", addr, err)
			return
		}
		errCh <- nil
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case err := <-errCh:
		return err

	case sig := <-sigCh:
		fmt.Printf("serve: shutting down on signal %s\n", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	return <-errCh
}

type neuteredFileSystem struct {
	fs http.FileSystem
}

func (nfs neuteredFileSystem) Open(path string) (http.File, error) {
	f, err := nfs.fs.Open(path)
	if err != nil {
		return nil, err
	}

	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, err
	}

	if info.IsDir() {
		index := filepath.Join(path, "index.html")
		indexFile, err := nfs.fs.Open(index)
		if err != nil {
			_ = f.Close()
			if errors.Is(err, os.ErrNotExist) {
				return nil, os.ErrNotExist
			}
			return nil, err
		}
		_ = indexFile.Close()
	}

	return f, nil
}
