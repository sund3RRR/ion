package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/alecthomas/kong"

	"github.com/sund3RRR/ion/internal/app/daemon"
	"github.com/sund3RRR/ion/internal/config"
	"github.com/sund3RRR/ion/internal/version"
)

type app struct {
	VersionFlag bool       `name:"version" help:"Print ION version and exit."`
	Version     VersionCmd `cmd:"" help:"Print ION version."`
	Install     InstallCmd `cmd:"" help:"Install a package."`
	Remove      RemoveCmd  `cmd:"" help:"Remove a package."`
	Update      UpdateCmd  `cmd:"" help:"Refresh packages."`
	Search      SearchCmd  `cmd:"" help:"Search packages."`
	List        ListCmd    `cmd:"" help:"List installed packages."`
	Profile     ProfileCmd `cmd:"" help:"Manage profiles."`
	Daemon      DaemonCmd  `cmd:"" help:"Run the ION daemon."`
}

type Runtime struct {
	cfg *config.Config
}

func Run(ctx context.Context, args []string) error {
	if isVersionFlag(args) {
		return printVersion()
	}

	root := app{}
	runtime := &Runtime{}

	parser, err := kong.New(&root,
		kong.Name("ion"),
		kong.Description("ION"),
		kong.UsageOnError(),
		kong.BindTo(ctx, (*context.Context)(nil)),
		kong.Bind(runtime),
	)
	if err != nil {
		return err
	}

	kctx, err := parser.Parse(args)
	if err != nil {
		return err
	}

	return kctx.Run()
}

func isVersionFlag(args []string) bool {
	return len(args) == 1 && args[0] == "--version"
}

func (r *Runtime) Config() (config.Config, error) {
	if r.cfg != nil {
		return *r.cfg, nil
	}

	cfg, err := config.Load()
	if err != nil {
		return config.Config{}, fmt.Errorf("load config: %w", err)
	}

	r.cfg = &cfg
	return cfg, nil
}

type InstallCmd struct{}
type RemoveCmd struct{}
type UpdateCmd struct{}
type SearchCmd struct{}
type ListCmd struct{}
type ProfileCmd struct{}
type VersionCmd struct{}

func (InstallCmd) Run(context.Context, *Runtime) error { return nil }
func (RemoveCmd) Run(context.Context, *Runtime) error  { return nil }
func (UpdateCmd) Run(context.Context, *Runtime) error  { return nil }
func (SearchCmd) Run(context.Context, *Runtime) error  { return nil }
func (ListCmd) Run(context.Context, *Runtime) error    { return nil }
func (ProfileCmd) Run(context.Context, *Runtime) error { return nil }

func (VersionCmd) Run() error {
	return printVersion()
}

func printVersion() error {
	_, err := fmt.Fprintln(os.Stdout, version.String())
	return err
}

type DaemonCmd struct {
	Socket string `help:"Override daemon Unix socket path."`
}

func (cmd DaemonCmd) Run(ctx context.Context, runtime *Runtime) error {
	cfg, err := runtime.Config()
	if err != nil {
		return err
	}

	return daemon.Run(ctx, cfg, daemon.Options{
		SocketPath: cmd.Socket,
		Version:    version.String(),
	})
}
