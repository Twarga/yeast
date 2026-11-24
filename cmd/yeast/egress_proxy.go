package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Twarga/yeast/internal/egress"
)

func newEgressProxyCmd() *cobra.Command {
	var (
		port      int
		allowlist []string
		allowFile string
		logPath   string
	)
	cmd := &cobra.Command{
		Use:    "egress-proxy",
		Short:  "Internal: run the allowlist egress proxy sidecar",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			allowed := append([]string{}, allowlist...)
			if allowFile != "" {
				f, err := os.Open(allowFile)
				if err != nil {
					return fmt.Errorf("open allowlist file: %w", err)
				}
				defer f.Close()
				sc := bufio.NewScanner(f)
				for sc.Scan() {
					if line := strings.TrimSpace(sc.Text()); line != "" && !strings.HasPrefix(line, "#") {
						allowed = append(allowed, line)
					}
				}
				if err := sc.Err(); err != nil {
					return fmt.Errorf("read allowlist file: %w", err)
				}
			}
			if len(allowed) == 0 {
				return fmt.Errorf("egress proxy requires at least one allowed domain")
			}
			proxy, err := egress.NewProxy(allowed, logPath)
			if err != nil {
				return err
			}
			defer proxy.Close()
			ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
			if err != nil {
				return fmt.Errorf("bind egress proxy: %w", err)
			}
			return egress.Serve(cmd.Context(), ln, proxy)
		},
	}
	cmd.Flags().IntVar(&port, "port", 0, "loopback port to bind")
	cmd.Flags().StringSliceVar(&allowlist, "allow", nil, "allowed domain (repeatable)")
	cmd.Flags().StringVar(&allowFile, "allowlist-file", "", "file with one allowed domain per line")
	cmd.Flags().StringVar(&logPath, "log", "", "decision log path")
	_ = cmd.MarkFlagRequired("port")
	return cmd
}
