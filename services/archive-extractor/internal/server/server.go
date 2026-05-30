// Package server provides a lightweight HTTP server for extracted sites.
package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// StaticServer serves files from a directory over HTTP.
type StaticServer struct {
	siteDir  string
	addr     string
	server   *http.Server
	listener net.Listener
	done     chan error
}

// Config configures the embedded HTTP server.
type Config struct {
	SiteDir string
	Port    string // defaults to 8080
	Addr    string // optional full listen addr (e.g. "127.0.0.1:0"); overrides Port when set
}

// NewStaticServer builds a StaticServer, applying defaults for missing fields.
func NewStaticServer(config *Config) *StaticServer {
	port := config.Port
	if port == "" {
		port = "8080"
	}

	addr := config.Addr
	if addr == "" {
		addr = "127.0.0.1:" + port
	}

	return &StaticServer{
		siteDir: config.SiteDir,
		addr:    addr,
		done:    make(chan error, 1),
	}
}

func containsDotPathSegment(pathValue string) bool {
	for _, segment := range strings.Split(path.Clean("/"+pathValue), "/") {
		if strings.HasPrefix(segment, ".") {
			return true
		}
	}

	return false
}

func (s *StaticServer) localPathForRequest(urlPath string) string {
	cleanPath := path.Clean("/" + urlPath)

	rel := strings.TrimPrefix(cleanPath, "/")
	if rel == "" {
		return filepath.Clean(s.siteDir)
	}

	return filepath.Join(filepath.Clean(s.siteDir), filepath.FromSlash(rel))
}

// Start begins serving in a background goroutine.
func (s *StaticServer) Start(ctx context.Context) error {
	if s.siteDir == "" {
		return errors.New("siteDir is required")
	}

	if _, err := os.Stat(s.siteDir); err != nil {
		return fmt.Errorf("invalid siteDir: %w", err)
	}

	root := http.Dir(filepath.Clean(s.siteDir))
	fileServer := http.FileServer(root)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)

			return
		}

		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

			return
		}

		if containsDotPathSegment(r.URL.Path) {
			http.NotFound(w, r)

			return
		}

		localPath := s.localPathForRequest(r.URL.Path)
		if info, statErr := os.Stat(localPath); statErr == nil && info.IsDir() {
			if _, indexErr := os.Stat(filepath.Join(localPath, "index.html")); indexErr != nil {
				http.NotFound(w, r)

				return
			}
		}

		fileServer.ServeHTTP(w, r)
	})

	server := &http.Server{
		Addr:              s.addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       30 * time.Second,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}

	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", s.addr, err)
	}

	s.server = server
	s.listener = listener

	go func() {
		serveErr := server.Serve(listener)
		if errors.Is(serveErr, http.ErrServerClosed) {
			s.done <- nil

			return
		}

		s.done <- serveErr
	}()

	slog.Info("Static server started", "addr", listener.Addr().String(), "site_dir", s.siteDir)

	return nil
}

// ListenerAddr returns the bound TCP address after Start succeeds.
func (s *StaticServer) ListenerAddr() string {
	if s.listener == nil {
		return ""
	}

	return s.listener.Addr().String()
}

// Wait blocks until the server exits.
func (s *StaticServer) Wait() error {
	if s.server == nil {
		return errors.New("server not started")
	}

	return <-s.done
}

// Stop gracefully shuts down the server.
func (s *StaticServer) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
}
