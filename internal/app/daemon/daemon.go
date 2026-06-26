package daemon

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"connectrpc.com/connect"

	"github.com/sund3RRR/ion/internal/config"
	ionv1 "github.com/sund3RRR/ion/pkg/pb/ion/v1"
	"github.com/sund3RRR/ion/pkg/pb/ion/v1/ionv1connect"
)

type Options struct {
	SocketPath string
	Version    string
}

func Run(ctx context.Context, cfg config.Config, opts Options) error {
	socketPath := opts.SocketPath
	if socketPath == "" {
		socketPath = cfg.Daemon.Socket.Path
	}
	if socketPath == "" {
		return fmt.Errorf("daemon socket path is empty")
	}

	listener, err := listenUnix(socketPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = os.Remove(socketPath)
	}()

	mux := http.NewServeMux()
	servicePath, handler := ionv1connect.NewDaemonServiceHandler(service{
		version: opts.Version,
	})
	mux.Handle(servicePath, handler)

	server := &http.Server{
		Handler: mux,
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return fmt.Errorf("shutdown daemon: %w", err)
		}

		if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve daemon: %w", err)
		}

		return nil
	case err := <-serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve daemon: %w", err)
		}

		return nil
	}
}

func listenUnix(path string) (net.Listener, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create daemon socket directory %q: %w", filepath.Dir(path), err)
	}

	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return nil, fmt.Errorf("daemon socket path %q exists and is not a socket", path)
		}

		conn, dialErr := net.DialTimeout("unix", path, 100*time.Millisecond)
		if dialErr == nil {
			_ = conn.Close()
			return nil, fmt.Errorf("daemon socket %q is already active", path)
		}

		if err := os.Remove(path); err != nil {
			return nil, fmt.Errorf("remove stale daemon socket %q: %w", path, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("inspect daemon socket %q: %w", path, err)
	}

	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("listen on daemon socket %q: %w", path, err)
	}

	if err := os.Chmod(path, 0o600); err != nil {
		_ = listener.Close()
		_ = os.Remove(path)
		return nil, fmt.Errorf("chmod daemon socket %q: %w", path, err)
	}

	return listener, nil
}

type service struct {
	version string
}

func (s service) Ping(context.Context, *connect.Request[ionv1.PingRequest]) (*connect.Response[ionv1.PingResponse], error) {
	return connect.NewResponse(&ionv1.PingResponse{
		Message: "pong",
	}), nil
}

func (s service) Version(context.Context, *connect.Request[ionv1.VersionRequest]) (*connect.Response[ionv1.VersionResponse], error) {
	version := s.version
	if version == "" {
		version = "dev"
	}

	return connect.NewResponse(&ionv1.VersionResponse{
		Version: version,
	}), nil
}
