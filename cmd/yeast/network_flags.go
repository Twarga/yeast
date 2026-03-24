package main

import (
	"fmt"
	"strings"
	"yeast/pkg/vm"

	"github.com/spf13/cobra"
)

var (
	networkModeFlag   string
	networkBridgeFlag string
)

func bindNetworkFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&networkModeFlag, "network-mode", vm.NetworkModeUser, "Network mode: user | private | bridge")
	cmd.Flags().StringVar(&networkBridgeFlag, "bridge", "", "Host bridge name (required when --network-mode=bridge)")
}

func networkOptionsFromFlags() (vm.NetworkOptions, error) {
	mode := strings.ToLower(strings.TrimSpace(networkModeFlag))
	if mode == "" {
		mode = vm.NetworkModeUser
	}

	bridge := strings.TrimSpace(networkBridgeFlag)

	switch mode {
	case vm.NetworkModeUser, vm.NetworkModePrivate:
		if bridge != "" {
			return vm.NetworkOptions{}, fmt.Errorf("--bridge can only be used with --network-mode=%s", vm.NetworkModeBridge)
		}
	case vm.NetworkModeBridge:
		if bridge == "" {
			return vm.NetworkOptions{}, fmt.Errorf("--network-mode=%s requires --bridge <bridge-name>", vm.NetworkModeBridge)
		}
	default:
		return vm.NetworkOptions{}, fmt.Errorf("invalid --network-mode %q (supported: %s, %s, %s)", mode, vm.NetworkModeUser, vm.NetworkModePrivate, vm.NetworkModeBridge)
	}

	return vm.NetworkOptions{
		Mode:   mode,
		Bridge: bridge,
	}, nil
}
