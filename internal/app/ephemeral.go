package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Twarga/yeast/internal/config"
	"github.com/Twarga/yeast/internal/guest"
	"github.com/Twarga/yeast/internal/provision/cloudinit"
	rtm "github.com/Twarga/yeast/internal/runtime"
)

// ephemeralVM is a throwaway QEMU instance booted from an existing disk —
// used by export sysprep/ISO builds where the user's real VM must never be
// mutated. cleanup() stops QEMU and removes the runtime dir.
type ephemeralVM struct {
	Instance rtm.RuntimeInstance
	SSHPort  int
	Dir      string
}

// ephemeralBoot creates a cloud-init seed (host SSH key, default yeast
// user), starts QEMU on diskPath, and waits for SSH readiness.
func (s *Service) ephemeralBoot(ctx context.Context, name, dir, diskPath string, memoryMiB, cpus int, online bool) (*ephemeralVM, func(), error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, nil, fmt.Errorf("create ephemeral dir: %w", err)
	}
	cleanup := func() {
		_ = s.runtime.Destroy(context.Background(), rtm.RuntimeInstance{Name: name, RuntimeDir: dir})
		_ = os.RemoveAll(dir)
	}

	userKey, err := s.discoverSSHKey()
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	if memoryMiB <= 0 {
		memoryMiB = 1024
	}
	if cpus <= 0 {
		cpus = 2
	}
	mac := deriveMACAddress("ephemeral", name, "management")

	userData, err := s.renderUserData(cloudinit.UserDataInput{
		Hostname:      name,
		Instance:      config.Instance{Name: name, User: config.DefaultUser, Sudo: config.DefaultSudo},
		AuthorizedKey: userKey,
	})
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	metaData, err := s.renderMetaData(cloudinit.MetaDataInput{Hostname: name})
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	networkConfig, err := s.renderNetworkConfig(cloudinit.NetworkConfigInput{
		ManagementInterfaceName: defaultManagementNIC,
		ManagementMACAddress:    mac,
	})
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	seed, err := s.createSeedISO(ctx, cloudinit.SeedInput{
		InstanceName: name, RuntimeDir: dir,
		UserData: userData, MetaData: metaData, NetworkConfig: networkConfig,
	})
	if err != nil {
		cleanup()
		return nil, nil, err
	}

	sshPort, err := freeLoopbackPort()
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	var egress *rtm.EgressPlan
	if !online {
		egress = &rtm.EgressPlan{Mode: "offline"}
	}
	plan := rtm.MachinePlan{
		Name:          name,
		RuntimeDir:    dir,
		LogPath:       filepath.Join(dir, "vm.log"),
		MemoryMiB:     memoryMiB,
		CPUs:          cpus,
		SeedImagePath: seed.ISOPath,
		Egress:        egress,
		Disk:          rtm.DiskPlan{DiskPath: diskPath},
		Networks: rtm.NetworkPlan{Management: rtm.ManagementNetworkPlan{
			SSHHost: "127.0.0.1", SSHPort: sshPort,
			InterfaceName: defaultManagementNIC, MACAddress: mac,
		}},
	}
	instance, err := s.runtime.Start(ctx, plan)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("start ephemeral VM: %w", err)
	}
	address, err := s.sshAddress("127.0.0.1", sshPort)
	if err == nil {
		err = s.waitForTCP(ctx, guest.ReadinessOptions{Address: address, Timeout: defaultReadinessTimeout})
	}
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("ephemeral VM not reachable: %w", err)
	}
	return &ephemeralVM{Instance: instance, SSHPort: sshPort, Dir: dir}, cleanup, nil
}
