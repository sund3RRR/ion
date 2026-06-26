package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sund3RRR/ion/internal/app/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := cli.Run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "ion: %v\n", err)
		os.Exit(1)
	}
}
