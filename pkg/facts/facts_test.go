package facts

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/m-horky/elk/internal/exec"
	"github.com/m-horky/elk/internal/fs"
)

// fakeRunner records the command it receives and returns a configured result.
type fakeRunner struct {
	run func(context.Context, exec.Command) (exec.Result, error)
}

// Run implements exec.Runner for tests.
func (runner fakeRunner) Run(ctx context.Context, command exec.Command) (exec.Result, error) {
	return runner.run(ctx, command)
}

// TestCollectPopulatesEverySource verifies best-effort collection across all parsers.
//
// Given readable os-release, virt-what, and lscpu sources, when facts are collected,
// then kernel, distribution, virtualization, and CPU facts are populated.
func TestCollectPopulatesEverySource(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	osReleasePath := filepath.Join(root, "os-release")
	writeFile(t, osReleasePath, "ID=rhel\nNAME=\"Red Hat Enterprise Linux\"\nVERSION_ID=\"10.2\"\n")

	runner := fakeRunner{run: func(_ context.Context, command exec.Command) (exec.Result, error) {
		switch command.Path {
		case "/test/virt-what":
			return exec.Result{Stdout: []byte("kvm\n")}, nil
		case "/test/lscpu":
			return exec.Result{Stdout: []byte(`{"lscpu":[` +
				`{"field":"CPU(s):","data":"8"},` +
				`{"field":"Socket(s):","data":"1"},` +
				`{"field":"Core(s) per socket:","data":"4"},` +
				`{"field":"Thread(s) per core:","data":"2"}]}`)}, nil
		default:
			return exec.Result{}, errors.New("unexpected command")
		}
	}}

	collected := collect(context.Background(), runner, fs.Filesystem{}, sourcePaths{
		osRelease: osReleasePath,
		lscpu:     "/test/lscpu",
		virtWhat:  "/test/virt-what",
	})

	if collected.SystemCertificateVersion == nil || *collected.SystemCertificateVersion != certificateVersion {
		t.Errorf("SystemCertificateVersion = %v, want %q", collected.SystemCertificateVersion, certificateVersion)
	}

	if collected.UnameSysname == nil || *collected.UnameSysname == "" {
		t.Error("UnameSysname is empty")
	}

	if collected.DistributionName == nil || *collected.DistributionName != "Red Hat Enterprise Linux" {
		t.Errorf("DistributionName = %v, want Red Hat Enterprise Linux", collected.DistributionName)
	}

	if collected.VirtIsGuest == nil || !*collected.VirtIsGuest {
		t.Errorf("VirtIsGuest = %v, want true", collected.VirtIsGuest)
	}

	if collected.LSCPU["cpu(s)"] != "8" {
		t.Errorf("LSCPU[\"cpu(s)\"] = %q, want 8", collected.LSCPU["cpu(s)"])
	}

	if collected.LSCPU["socket(s)"] != "1" {
		t.Errorf("LSCPU[\"socket(s)\"] = %q, want 1", collected.LSCPU["socket(s)"])
	}
}

// TestCollectToleratesSourceFailures verifies individual sources do not abort collection.
//
// Given failing command sources and a missing os-release file, when facts are collected,
// then the certificate version is still populated and failed sources stay nil.
func TestCollectToleratesSourceFailures(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	runner := fakeRunner{run: func(context.Context, exec.Command) (exec.Result, error) {
		return exec.Result{}, errors.New("source unavailable")
	}}

	collected := collect(context.Background(), runner, fs.Filesystem{}, sourcePaths{
		osRelease: filepath.Join(root, "missing"),
		lscpu:     "/test/lscpu",
		virtWhat:  "/test/virt-what",
	})

	if collected.SystemCertificateVersion == nil {
		t.Fatal("SystemCertificateVersion is nil")
	}

	if collected.DistributionName != nil {
		t.Errorf("DistributionName = %v, want nil", collected.DistributionName)
	}

	if collected.VirtIsGuest != nil {
		t.Errorf("VirtIsGuest = %v, want nil", collected.VirtIsGuest)
	}

	if collected.LSCPU != nil {
		t.Errorf("LSCPU = %v, want nil", collected.LSCPU)
	}
}

// TestCollectBoundsCommandLifetime verifies command sources receive a deadline.
//
// Given a runner that inspects the context, when facts are collected,
// then every command invocation is bounded by commandTimeout and inherits
// cancellation from the caller context.
func TestCollectBoundsCommandLifetime(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	var deadlines []time.Duration

	runner := fakeRunner{run: func(ctx context.Context, _ exec.Command) (exec.Result, error) {
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Error("command context has no deadline")
		} else {
			deadlines = append(deadlines, time.Until(deadline))
		}

		return exec.Result{}, errors.New("source unavailable")
	}}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	collect(ctx, runner, fs.Filesystem{}, sourcePaths{
		osRelease: filepath.Join(root, "missing"),
		lscpu:     "/test/lscpu",
		virtWhat:  "/test/virt-what",
	})

	if len(deadlines) != 2 {
		t.Fatalf("command invocations = %d, want 2", len(deadlines))
	}

	for _, remaining := range deadlines {
		if remaining > commandTimeout || remaining <= 0 {
			t.Errorf("command deadline remaining = %v, want within (0, %v]", remaining, commandTimeout)
		}
	}
}

// TestCollectReturnsWhenContextIsCanceled verifies command cancellation propagates.
//
// Given a runner that blocks until its context ends, when collection is bounded
// by a short deadline, then collection returns promptly with command facts unset.
func TestCollectReturnsWhenContextIsCanceled(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	runner := fakeRunner{run: func(ctx context.Context, _ exec.Command) (exec.Result, error) {
		<-ctx.Done()

		return exec.Result{}, ctx.Err()
	}}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	started := time.Now()
	collected := collect(ctx, runner, fs.Filesystem{}, sourcePaths{
		osRelease: filepath.Join(root, "missing"),
		lscpu:     "/test/lscpu",
		virtWhat:  "/test/virt-what",
	})

	if elapsed := time.Since(started); elapsed > time.Second {
		t.Errorf("collect() took %v, want cancellation within one second", elapsed)
	}

	if collected.VirtIsGuest != nil {
		t.Errorf("VirtIsGuest = %v, want nil", collected.VirtIsGuest)
	}

	if collected.LSCPU != nil {
		t.Errorf("LSCPU = %v, want nil", collected.LSCPU)
	}
}

// writeFile writes test fixture content to path.
func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
