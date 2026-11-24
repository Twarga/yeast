// sdk-basic shows the minimal yeastd SDK flow: connect over the unix
// socket, register the current project, print status, and stream events
// while an action runs.
//
// Requires `yeast daemon` to be running.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Twarga/yeast/pkg/yeast"
)

func main() {
	ctx := context.Background()

	client, err := yeast.Connect("") // default $XDG_RUNTIME_DIR/yeast/yeastd.sock
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect:", err)
		os.Exit(1)
	}

	root, _ := os.Getwd()
	id, err := client.RegisterProject(ctx, root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "register:", err)
		os.Exit(1)
	}
	fmt.Println("project:", id)

	status, err := client.Status(ctx, id)
	if err != nil {
		fmt.Fprintln(os.Stderr, "status:", err)
		os.Exit(1)
	}
	for _, inst := range status.Instances {
		fmt.Printf("  %-16s %-9s ssh:%d\n", inst.Name, inst.Status, inst.SSHPort)
	}

	// Stream events for 10s — run `yeast up` elsewhere to see them flow.
	events, errs := client.Events(ctx, id)
	timeout := time.After(10 * time.Second)
	for {
		select {
		case e := <-events:
			fmt.Printf("event %-28s %s\n", e.Name, e.Message)
		case err := <-errs:
			if err != nil {
				fmt.Fprintln(os.Stderr, "events:", err)
			}
			return
		case <-timeout:
			return
		}
	}
}
