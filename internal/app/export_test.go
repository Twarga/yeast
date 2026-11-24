package app

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Twarga/yeast/internal/guest"
	"github.com/Twarga/yeast/internal/provision/cloudinit"
	provssh "github.com/Twarga/yeast/internal/provision/ssh"
	rtm "github.com/Twarga/yeast/internal/runtime"
	"github.com/Twarga/yeast/internal/state"
)

type fakeExportRuntime struct {
	fakeLiveRuntime
	started     []rtm.MachinePlan
	exportedSrc string
	exportedDst string
	clonedSrc   string
	blankPath   string
	startErr    error
}

func (f *fakeExportRuntime) Start(ctx context.Context, plan rtm.MachinePlan) (rtm.RuntimeInstance, error) {
	f.started = append(f.started, plan)
	if f.startErr != nil {
		return rtm.RuntimeInstance{}, f.startErr
	}
	return rtm.RuntimeInstance{Name: plan.Name, RuntimeDir: plan.RuntimeDir, PID: 9000 + len(f.started)}, nil
}
func (f *fakeExportRuntime) Stop(ctx context.Context, i rtm.RuntimeInstance, _ time.Duration) error {
	return nil
}
func (f *fakeExportRuntime) Inspect(ctx context.Context, i rtm.RuntimeInstance) (rtm.ProcessInfo, error) {
	if i.PID == 4242 {
		return rtm.ProcessInfo{PID: i.PID, State: rtm.ProcessStateRunning}, nil
	}
	return rtm.ProcessInfo{PID: i.PID, State: rtm.ProcessStateStopped}, nil
}
func (f *fakeExportRuntime) Destroy(ctx context.Context, i rtm.RuntimeInstance) error { return nil }
func (f *fakeExportRuntime) ExportDisk(ctx context.Context, src, dst string) error {
	f.exportedSrc, f.exportedDst = src, dst
	return os.WriteFile(dst, []byte("qcow2-export"), 0o644)
}
func (f *fakeExportRuntime) CloneDisk(ctx context.Context, src, dstDir string) (string, error) {
	f.clonedSrc = src
	p := filepath.Join(dstDir, "clone.qcow2")
	return p, os.WriteFile(p, []byte("clone"), 0o644)
}
func (f *fakeExportRuntime) CreateBlankDisk(ctx context.Context, path, size string) error {
	f.blankPath = path
	return os.WriteFile(path, []byte("blank"), 0o644)
}

// fakeStreamTransport captures Run commands and streams canned bytes.
type fakeStreamTransport struct {
	runs      []provssh.RunRequest
	streamCmd string
	downloads []provssh.DownloadRequest
	streamOut []byte
}

func (f *fakeStreamTransport) Run(ctx context.Context, req provssh.RunRequest) (provssh.RunResult, error) {
	f.runs = append(f.runs, req)
	return provssh.RunResult{ExitCode: 0}, nil
}
func (f *fakeStreamTransport) Upload(ctx context.Context, req provssh.UploadRequest) error {
	return nil
}
func (f *fakeStreamTransport) Download(ctx context.Context, req provssh.DownloadRequest) error {
	f.downloads = append(f.downloads, req)
	return os.WriteFile(req.Destination, []byte("iso-bytes"), 0o644)
}
func (f *fakeStreamTransport) Stream(ctx context.Context, req provssh.StreamRequest) error {
	f.streamCmd = req.Command
	_, err := io.Copy(req.Stdout, strings.NewReader(string(f.streamOut)))
	return err
}

func stubEphemeralDeps(s *Service) {
	s.discoverSSHKey = func() (string, error) { return "ssh-ed25519 AAAA", nil }
	s.createSeedISO = func(ctx context.Context, in cloudinit.SeedInput) (cloudinit.SeedResult, error) {
		return cloudinit.SeedResult{ISOPath: filepath.Join(in.RuntimeDir, "seed.iso")}, nil
	}
	s.waitForTCP = func(ctx context.Context, o guest.ReadinessOptions) error { return nil }
}

func newExportService(t *testing.T, running bool) (*Service, string, *fakeExportRuntime, *fakeStreamTransport, state.InstanceState) {
	t.Helper()
	service, root, metadata, _ := newSnapshotServiceWithStoppedInstance(t)
	fake := &fakeExportRuntime{}
	service.runtime = fake
	transport := &fakeStreamTransport{streamOut: []byte("rootfs-tar")}
	service.provisionTransport = transport
	stubEphemeralDeps(service)

	paths, instRtDir := seedStoppedWithPlan(t, service, root, metadata)
	st, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		t.Fatal(err)
	}
	inst := st.Instances["web"]
	if running {
		inst.PID = 4242
		inst.Status = "running"
		inst.SSHPort = 2222
	}
	st.Instances["web"] = inst
	if err := state.Save(paths.StateFile, st); err != nil {
		t.Fatal(err)
	}
	inst.RuntimeDir = instRtDir
	return service, root, fake, transport, inst
}

func TestExportQCOW2StoppedFlatten(t *testing.T) {
	service, root, fake, _, _ := newExportService(t, false)
	out := filepath.Join(t.TempDir(), "web.qcow2")
	res, err := service.Export(context.Background(), ExportOptions{
		GuestTargetOptions: GuestTargetOptions{ProjectRoot: root, Target: "web"},
		Format:             "qcow2",
		Output:             out,
	})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if fake.exportedSrc == "" || !strings.HasSuffix(fake.exportedSrc, "disk.qcow2") {
		t.Fatalf("ExportDisk not called on instance disk: %q", fake.exportedSrc)
	}
	if res.Format != "qcow2" || res.Bytes == 0 || res.SHA256 == "" {
		t.Fatalf("bad result: %+v", res)
	}
}

func TestExportQCOW2RunningRejected(t *testing.T) {
	service, root, _, _, _ := newExportService(t, true)
	_, err := service.Export(context.Background(), ExportOptions{
		GuestTargetOptions: GuestTargetOptions{ProjectRoot: root, Target: "web"},
		Format:             "qcow2",
		Output:             filepath.Join(t.TempDir(), "x.qcow2"),
	})
	if err == nil || !strings.Contains(err.Error(), "running") {
		t.Fatalf("expected running precondition, got %v", err)
	}
}

func TestExportQCOW2SysprepClones(t *testing.T) {
	service, root, fake, transport, _ := newExportService(t, false)
	out := filepath.Join(t.TempDir(), "clean.qcow2")
	_, err := service.Export(context.Background(), ExportOptions{
		GuestTargetOptions: GuestTargetOptions{ProjectRoot: root, Target: "web"},
		Format:             "qcow2",
		Output:             out,
		Sysprep:            true,
	})
	if err != nil {
		t.Fatalf("Export sysprep: %v", err)
	}
	if fake.clonedSrc == "" {
		t.Fatal("sysprep must clone the disk")
	}
	if !strings.HasPrefix(fake.exportedSrc, filepath.Dir(fake.clonedSrc)) && !strings.Contains(fake.exportedSrc, "clone") {
		t.Fatalf("export must use the clone, got %q", fake.exportedSrc)
	}
	// sysprep script must have run on the ephemeral VM
	found := false
	for _, r := range transport.runs {
		if strings.Contains(r.Command, "machine-id") && strings.Contains(r.Command, "ssh_host_") {
			found = true
		}
	}
	if !found {
		t.Fatalf("sysprep script did not run: %+v", transport.runs)
	}
	if len(fake.started) == 0 {
		t.Fatal("ephemeral clone VM never booted")
	}
}

func TestExportRootfsRunningStreams(t *testing.T) {
	service, root, _, transport, _ := newExportService(t, true)
	out := filepath.Join(t.TempDir(), "root.tar.zst")
	res, err := service.Export(context.Background(), ExportOptions{
		GuestTargetOptions: GuestTargetOptions{ProjectRoot: root, Target: "web"},
		Format:             "rootfs",
		Output:             out,
	})
	if err != nil {
		t.Fatalf("Export rootfs: %v", err)
	}
	if !strings.Contains(transport.streamCmd, "tar") || !strings.Contains(transport.streamCmd, "--exclude=./proc") {
		t.Fatalf("stream command wrong: %q", transport.streamCmd)
	}
	data, err := os.ReadFile(out)
	if err != nil || len(data) < 4 {
		t.Fatalf("output missing: %v", err)
	}
	// zstd magic \x28\xb5\x2f\xfd
	if data[0] != 0x28 || data[1] != 0xb5 || data[2] != 0x2f || data[3] != 0xfd {
		t.Fatalf("output is not zstd: %x", data[:4])
	}
	if res.Bytes == 0 {
		t.Fatal("result bytes not populated")
	}
}

func TestExportISORunsBuilderSteps(t *testing.T) {
	service, root, fake, transport, _ := newExportService(t, false)
	out := filepath.Join(t.TempDir(), "live.iso")
	res, err := service.Export(context.Background(), ExportOptions{
		GuestTargetOptions: GuestTargetOptions{ProjectRoot: root, Target: "web"},
		Format:             "iso",
		Output:             out,
	})
	if err != nil {
		t.Fatalf("Export iso: %v", err)
	}
	var joined string
	for _, r := range transport.runs {
		joined += r.Command + "\n"
	}
	for _, want := range []string{"mksquashfs", "grub-mkrescue", "apt-get install"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("iso pipeline missing %q", want)
		}
	}
	if len(transport.downloads) != 1 {
		t.Fatalf("expected iso download, got %d", len(transport.downloads))
	}
	if res.Builder != "debian-live" {
		t.Fatalf("builder not reported: %+v", res)
	}
	if fake.clonedSrc == "" || len(fake.started) == 0 {
		t.Fatal("iso build must run on an ephemeral clone")
	}
}

func TestBootISOPlanAndScreenshot(t *testing.T) {
	root := t.TempDir()
	service := NewService()
	service.resolveYeastHome = func() (string, error) { return filepath.Join(root, "yh"), nil }
	fake := &fakeExportRuntime{}
	service.runtime = fake
	stubEphemeralDeps(service)

	iso := filepath.Join(root, "test.iso")
	if err := os.WriteFile(iso, []byte("iso"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := service.BootISO(context.Background(), BootISOOptions{
		ISOPath: iso, DiskSize: "10G", Remove: true,
	})
	if err != nil {
		t.Fatalf("BootISO: %v", err)
	}
	if len(fake.started) != 1 {
		t.Fatalf("expected 1 boot, got %d", len(fake.started))
	}
	plan := fake.started[0]
	if plan.BootISOPath != iso {
		t.Fatalf("plan missing BootISOPath: %+v", plan)
	}
	if fake.blankPath == "" {
		t.Fatal("--disk should create a scratch disk")
	}
	if !res.Removed {
		t.Fatal("--rm must mark result removed")
	}
}
