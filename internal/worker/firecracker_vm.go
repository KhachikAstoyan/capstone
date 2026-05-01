//go:build linux

package worker

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
)

const (
	fcVsockPort    = 52000
	fcVsockFile    = "fc.vsock" // path inside chroot root
	fcSnapshotFile = "snapshot" // path inside chroot root
	fcMemFile      = "mem"      // path inside chroot root
)

// workspaceDir returns the absolute host path to the jailer chroot root
// for a given VM ID: {ChrootBase}/firecracker/{vmID}/root
func workspaceDir(cfg FCConfig, vmID string) string {
	return filepath.Join(cfg.ChrootBase, "firecracker", vmID, "root")
}

// fcVM wraps a running Firecracker Machine and its derived paths.
type fcVM struct {
	machine   *firecracker.Machine
	cancelFn  context.CancelFunc
	vsockPath string // absolute host path to vsock UDS
	wsDir     string // absolute host path to jailer chroot root
	log       *zap.Logger
}

// ── Boot a fresh VM for snapshot capture ─────────────────────────────────────

// bootFreshVM starts a new VM from rootfsPath, fully booted and ready for
// snapshot capture. The caller must call kill() + cleanup() when done.
func bootFreshVM(cfg FCConfig, vmID, rootfsPath string, log *zap.Logger) (*fcVM, error) {
	ctx, cancel := context.WithCancel(context.Background())

	uid := cfg.JailerUID
	gid := cfg.JailerGID
	numa := 0

	fcCfg := firecracker.Config{
		KernelImagePath: cfg.KernelPath,
		KernelArgs:      "console=ttyS0 reboot=k panic=1 pci=off nomodules quiet",
		Drives: []models.Drive{{
			DriveID:      firecracker.String("rootfs"),
			PathOnHost:   firecracker.String(rootfsPath),
			IsRootDevice: firecracker.Bool(true),
			IsReadOnly:   firecracker.Bool(false),
		}},
		MachineCfg: models.MachineConfiguration{
			VcpuCount:  firecracker.Int64(cfg.vcpu()),
			MemSizeMib: firecracker.Int64(cfg.memMB()),
		},
		VsockDevices: []firecracker.VsockDevice{{
			Path: fcVsockFile,
			CID:  3,
		}},
		JailerCfg: &firecracker.JailerConfig{
			UID:            &uid,
			GID:            &gid,
			NumaNode:       &numa,
			ID:             vmID,
			ExecFile:       cfg.FirecrackerBin,
			JailerBinary:   cfg.JailerBin,
			ChrootBaseDir:  cfg.ChrootBase,
			ChrootStrategy: firecracker.NewNaiveChrootStrategy(cfg.KernelPath),
			Stdout:         os.Stderr,
			Stderr:         os.Stderr,
		},
	}

	machine, err := firecracker.NewMachine(ctx, fcCfg, firecracker.WithLogger(silentLogger()))
	if err != nil {
		cancel()
		return nil, fmt.Errorf("new machine: %w", err)
	}
	if err := machine.Start(ctx); err != nil {
		cancel()
		machine.StopVMM() //nolint:errcheck
		return nil, fmt.Errorf("start machine: %w", err)
	}

	wsDir := workspaceDir(cfg, vmID)
	return &fcVM{
		machine:   machine,
		cancelFn:  cancel,
		vsockPath: filepath.Join(wsDir, fcVsockFile),
		wsDir:     wsDir,
		log:       log,
	}, nil
}

// ── Restore a VM from snapshot ────────────────────────────────────────────────

// restoreVM loads a snapshot into a new jailer-managed Firecracker instance.
// snapFile and memFile are absolute host paths to the snapshot pair.
func restoreVM(cfg FCConfig, vmID, snapFile, memFile string, log *zap.Logger) (*fcVM, error) {
	// Pre-populate the chroot with snapshot files before the jailer runs.
	wsDir := workspaceDir(cfg, vmID)
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir workspace: %w", err)
	}
	if err := copyFile(snapFile, filepath.Join(wsDir, fcSnapshotFile)); err != nil {
		return nil, fmt.Errorf("copy snapshot: %w", err)
	}
	if err := copyFile(memFile, filepath.Join(wsDir, fcMemFile)); err != nil {
		return nil, fmt.Errorf("copy mem: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	uid := cfg.JailerUID
	gid := cfg.JailerGID
	numa := 0

	fcCfg := firecracker.Config{
		// Snapshot paths are as seen from INSIDE the chroot (absolute from chroot root).
		Snapshot: firecracker.SnapshotConfig{
			SnapshotPath: "/" + fcSnapshotFile,
			MemFilePath:  "/" + fcMemFile,
			ResumeVM:     true,
		},
		VsockDevices: []firecracker.VsockDevice{{
			Path: fcVsockFile,
			CID:  3,
		}},
		// Skip host-side stat validation: snapshot files live inside the chroot,
		// not at their chroot-relative paths on the host filesystem.
		DisableValidation: true,
		JailerCfg: &firecracker.JailerConfig{
			UID:           &uid,
			GID:           &gid,
			NumaNode:      &numa,
			ID:            vmID,
			ExecFile:      cfg.FirecrackerBin,
			JailerBinary:  cfg.JailerBin,
			ChrootBaseDir: cfg.ChrootBase,
			// No-op: snapshot files are already in the chroot from the pre-copy above.
			ChrootStrategy: emptyChrootStrategy{},
			Stdout:         os.Stderr,
			Stderr:         os.Stderr,
		},
	}

	machine, err := firecracker.NewMachine(ctx, fcCfg, firecracker.WithLogger(silentLogger()))
	if err != nil {
		cancel()
		cleanupVM(cfg, vmID)
		return nil, fmt.Errorf("new machine: %w", err)
	}
	if err := machine.Start(ctx); err != nil {
		cancel()
		machine.StopVMM() //nolint:errcheck
		cleanupVM(cfg, vmID)
		return nil, fmt.Errorf("start snapshot machine: %w", err)
	}

	return &fcVM{
		machine:   machine,
		cancelFn:  cancel,
		vsockPath: filepath.Join(wsDir, fcVsockFile),
		wsDir:     wsDir,
		log:       log,
	}, nil
}

// ── VM methods ────────────────────────────────────────────────────────────────

// waitForAgent polls vsock until the fc-agent accepts the Firecracker handshake.
func (vm *fcVM) waitForAgent(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if conn, err := vm.dialVsock(); err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("fc-agent not ready after %v", timeout)
}

// takeSnapshot pauses the VM, creates a full snapshot, copies the files to
// snapDst/memDst, then resumes.
func (vm *fcVM) takeSnapshot(snapDst, memDst string) error {
	ctx := context.Background()
	if err := vm.machine.PauseVM(ctx); err != nil {
		return fmt.Errorf("pause VM: %w", err)
	}
	// Paths are chroot-relative; files land at {wsDir}/{name} on the host.
	if err := vm.machine.CreateSnapshot(ctx, fcMemFile, fcSnapshotFile); err != nil {
		return fmt.Errorf("create snapshot: %w", err)
	}
	if err := vm.machine.ResumeVM(ctx); err != nil {
		vm.log.Warn("resume after snapshot failed", zap.Error(err))
	}
	if err := copyFile(filepath.Join(vm.wsDir, fcSnapshotFile), snapDst); err != nil {
		return fmt.Errorf("copy snapshot out: %w", err)
	}
	return copyFile(filepath.Join(vm.wsDir, fcMemFile), memDst)
}

// dialVsock connects to the fc-agent using Firecracker's vsock-UDS proxy.
// Protocol: send "CONNECT {port}\n", read "OK {port}\n".
func (vm *fcVM) dialVsock() (net.Conn, error) {
	conn, err := net.DialTimeout("unix", vm.vsockPath, 2*time.Second)
	if err != nil {
		return nil, err
	}
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	fmt.Fprintf(conn, "CONNECT %d\n", fcVsockPort)
	buf := make([]byte, 32)
	n, err := conn.Read(buf)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("vsock handshake: %w", err)
	}
	conn.SetDeadline(time.Time{})
	if n < 2 || string(buf[:2]) != "OK" {
		conn.Close()
		return nil, fmt.Errorf("vsock handshake unexpected: %q", string(buf[:n]))
	}
	return conn, nil
}

// kill stops the Firecracker process.
func (vm *fcVM) kill() {
	vm.cancelFn()
	vm.machine.StopVMM() //nolint:errcheck
}

// cleanup removes the VM's chroot directory.
func (vm *fcVM) cleanup() {
	cleanupVM(FCConfig{ChrootBase: filepath.Dir(filepath.Dir(filepath.Dir(vm.wsDir)))},
		filepath.Base(filepath.Dir(vm.wsDir)))
}

// ── helpers ────────────────────────────────────────────────────────────────────

func cleanupVM(cfg FCConfig, vmID string) {
	vmDir := filepath.Join(cfg.ChrootBase, "firecracker", vmID)
	os.RemoveAll(vmDir) //nolint:errcheck
}

// emptyChrootStrategy is a no-op HandlersAdapter used when files are already
// in the chroot (snapshot restore path).
type emptyChrootStrategy struct{}

func (emptyChrootStrategy) AdaptHandlers(_ *firecracker.Handlers) error { return nil }

// copyFile copies src to dst with a hardlink fast-path (same filesystem).
func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	// Try hardlink first (zero-copy, same filesystem).
	if err := os.Link(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// silentLogger returns a logrus entry that discards all output, used to
// suppress the SDK's own logging (we use zap instead).
func silentLogger() *logrus.Entry {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return logrus.NewEntry(l)
}
