package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/Twarga/yeast/internal/app"
	"github.com/Twarga/yeast/internal/daemon"

	"github.com/spf13/cobra"
)

func newDaemonCmd(service *app.Service) *cobra.Command {
	var socketPath string
	var tcpAddr string
	var token string
	var idleTimeout time.Duration

	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Run the local yeastd service (unix socket HTTP API + VNC proxy)",
		RunE: func(cmd *cobra.Command, args []string) error {
			srv, err := daemon.New(service, daemon.Options{
				SocketPath:  socketPath,
				TCPAddr:     tcpAddr,
				Token:       token,
				IdleTimeout: idleTimeout,
			})
			if err != nil {
				return err
			}
			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()
			fmt.Fprintf(cmd.ErrOrStderr(), "yeastd listening on %s\n", srv.SocketPath())
			if tcpAddr != "" {
				fmt.Fprintf(cmd.ErrOrStderr(), "yeastd TCP listener on %s (bearer auth)\n", tcpAddr)
			}
			return srv.Serve(ctx)
		},
	}

	cmd.Flags().StringVar(&socketPath, "socket", "", "Unix socket path (default $XDG_RUNTIME_DIR/yeast/yeastd.sock)")
	cmd.Flags().StringVar(&tcpAddr, "tcp", "", "Optional loopback TCP listener, e.g. 127.0.0.1:7357 (requires --token)")
	cmd.Flags().StringVar(&token, "token", "", "Bearer token required for the TCP listener")
	cmd.Flags().DurationVar(&idleTimeout, "idle-timeout", 0, "Exit after this much inactivity (e.g. 30m)")
	return cmd
}
