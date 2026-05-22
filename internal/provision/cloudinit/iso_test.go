package cloudinit

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCreateSeedISOWritesInputsAndInvokesBuilder(t *testing.T) {
	previousLookPath := lookPath
	previousRun := runISOCommand
	defer func() {
		lookPath = previousLookPath
		runISOCommand = previousRun
	}()

	lookPath = func(file string) (string, error) {
		if file == "genisoimage" {
			return "/usr/bin/genisoimage", nil
		}
		return "", errors.New("not found")
	}

	var gotName string
	var gotArgs []string
	runISOCommand = func(_ context.Context, name string, args ...string) error {
		gotName = name
		gotArgs = append([]string(nil), args...)
		return nil
	}

	root := t.TempDir()
	result, err := CreateSeedISO(context.Background(), SeedInput{
		InstanceName:  "web",
		RuntimeDir:    filepath.Join(root, "instances", "web"),
		UserData:      "#cloud-config\nhostname: web\n",
		MetaData:      "instance-id: web\nlocal-hostname: web\n",
		NetworkConfig: "version: 2\nethernets: {}\n",
	})
	if err != nil {
		t.Fatalf("CreateSeedISO returned error: %v", err)
	}

	if gotName != "genisoimage" {
		t.Fatalf("unexpected builder: got %q", gotName)
	}
	wantArgs := []string{
		"-output", filepath.Join(root, "instances", "web", "seed.iso"),
		"-volid", "cidata",
		"-joliet",
		"-rock",
		filepath.Join(root, "instances", "web", "user-data"),
		filepath.Join(root, "instances", "web", "meta-data"),
		filepath.Join(root, "instances", "web", "network-config"),
	}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("unexpected args:\n got: %#v\nwant: %#v", gotArgs, wantArgs)
	}

	userData, err := os.ReadFile(result.UserDataPath)
	if err != nil {
		t.Fatalf("read user-data: %v", err)
	}
	if string(userData) != "#cloud-config\nhostname: web\n" {
		t.Fatalf("unexpected user-data content: %q", string(userData))
	}

	metaData, err := os.ReadFile(result.MetaDataPath)
	if err != nil {
		t.Fatalf("read meta-data: %v", err)
	}
	if string(metaData) != "instance-id: web\nlocal-hostname: web\n" {
		t.Fatalf("unexpected meta-data content: %q", string(metaData))
	}

	if result.ISOPath != filepath.Join(root, "instances", "web", "seed.iso") {
		t.Fatalf("unexpected iso path: %q", result.ISOPath)
	}
	networkConfig, err := os.ReadFile(result.NetworkConfigPath)
	if err != nil {
		t.Fatalf("read network-config: %v", err)
	}
	if string(networkConfig) != "version: 2\nethernets: {}\n" {
		t.Fatalf("unexpected network-config content: %q", string(networkConfig))
	}
}

func TestCreateSeedISOMissingBuilderReturnsClearError(t *testing.T) {
	previousLookPath := lookPath
	defer func() {
		lookPath = previousLookPath
	}()

	lookPath = func(string) (string, error) {
		return "", errors.New("not found")
	}

	root := t.TempDir()
	_, err := CreateSeedISO(context.Background(), SeedInput{
		InstanceName: "web",
		RuntimeDir:   filepath.Join(root, "instances", "web"),
		UserData:     "#cloud-config\nhostname: web\n",
		MetaData:     "instance-id: web\nlocal-hostname: web\n",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNoISOBuilder) {
		t.Fatalf("expected ErrNoISOBuilder, got %v", err)
	}
	if !strings.Contains(err.Error(), "genisoimage") || !strings.Contains(err.Error(), "mkisofs") {
		t.Fatalf("expected installation hint, got %q", err)
	}
}

func TestDiscoverISOBuilderFallsBackToMkisofs(t *testing.T) {
	previousLookPath := lookPath
	defer func() {
		lookPath = previousLookPath
	}()

	lookPath = func(file string) (string, error) {
		if file == "mkisofs" {
			return "/usr/bin/mkisofs", nil
		}
		return "", errors.New("not found")
	}

	builder, err := discoverISOBuilder()
	if err != nil {
		t.Fatalf("discoverISOBuilder returned error: %v", err)
	}
	if builder.Name != "mkisofs" {
		t.Fatalf("expected mkisofs fallback, got %q", builder.Name)
	}
}
