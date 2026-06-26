package daemon

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/sund3RRR/ion/internal/config"
	ionv1 "github.com/sund3RRR/ion/pkg/pb/ion/v1"
	"github.com/sund3RRR/ion/pkg/pb/ion/v1/ionv1connect"
)

func TestRunPing(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "iond.sock")
	cfg := config.Config{
		Daemon: config.DaemonConfig{
			Socket: config.SocketConfig{
				Path: socketPath,
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(ctx, cfg, Options{Version: "test"})
	}()

	client := ionv1connect.NewDaemonServiceClient(unixHTTPClient(socketPath), "http://ion")

	var ping *connect.Response[ionv1.PingResponse]
	var err error
	deadline := time.Now().Add(2 * time.Second)
	for {
		ping, err = client.Ping(context.Background(), connect.NewRequest(&ionv1.PingRequest{}))
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatalf("Ping() error = %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	if got := ping.Msg.GetMessage(); got != "pong" {
		t.Fatalf("Ping().message = %q, want %q", got, "pong")
	}

	version, err := client.Version(context.Background(), connect.NewRequest(&ionv1.VersionRequest{}))
	if err != nil {
		cancel()
		t.Fatalf("Version() error = %v", err)
	}
	if got := version.Msg.GetVersion(); got != "test" {
		cancel()
		t.Fatalf("Version().version = %q, want %q", got, "test")
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("daemon did not stop after context cancellation")
	}

	if _, err := os.Stat(socketPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("socket exists after shutdown: %v", err)
	}
}

func unixHTTPClient(socketPath string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var dialer net.Dialer
				return dialer.DialContext(ctx, "unix", socketPath)
			},
		},
	}
}
