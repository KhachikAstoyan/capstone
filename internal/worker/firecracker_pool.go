//go:build linux

package worker

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// SnapshotPair holds the host paths of a captured VM snapshot.
type SnapshotPair struct {
	SnapshotFile string
	MemFile      string
}

// SnapshotPool pre-boots one VM per language, captures a full snapshot, and
// serves that snapshot for every subsequent job execution.
//
// The pool is immutable after Init: snapshots are created once at worker
// startup and reused for the lifetime of the process. The snapshot itself
// is never modified; each restored VM gets its own chroot with copied files.
type SnapshotPool struct {
	mu        sync.RWMutex
	snapshots map[string]SnapshotPair // language → snapshot pair
	cfg       FCConfig
	snapsDir  string // directory where snapshot files are kept
	log       *zap.Logger
}

// NewSnapshotPool creates an empty pool.
// Call Init to boot and snapshot each language.
func NewSnapshotPool(cfg FCConfig, snapsDir string, log *zap.Logger) *SnapshotPool {
	return &SnapshotPool{
		snapshots: make(map[string]SnapshotPair),
		cfg:       cfg,
		snapsDir:  snapsDir,
		log:       log,
	}
}

// Init boots a fresh VM for each language in rootfsByLang, waits for the
// fc-agent to be ready, captures a full snapshot, then kills the boot VM.
// rootfsByLang maps language key (e.g. "python") → host path to ext4 rootfs.
func (p *SnapshotPool) Init(rootfsByLang map[string]string) error {
	for lang, rootfs := range rootfsByLang {
		if err := p.snapshotLanguage(lang, rootfs); err != nil {
			return fmt.Errorf("snapshot %s: %w", lang, err)
		}
	}
	return nil
}

func (p *SnapshotPool) snapshotLanguage(lang, rootfsPath string) error {
	log := p.log.With(zap.String("language", lang))
	log.Info("booting VM for snapshot capture")

	vmID := "boot-" + lang + "-" + uuid.New().String()[:8]
	vm, err := bootFreshVM(p.cfg, vmID, rootfsPath, log)
	if err != nil {
		return fmt.Errorf("boot VM: %w", err)
	}
	defer func() {
		vm.kill()
		vm.cleanup()
	}()

	log.Info("waiting for fc-agent")
	if err := vm.waitForAgent(30 * time.Second); err != nil {
		return fmt.Errorf("agent not ready: %w", err)
	}
	log.Info("fc-agent ready, capturing snapshot")

	snapFile := filepath.Join(p.snapsDir, lang+".snap")
	memFile := filepath.Join(p.snapsDir, lang+".mem")

	if err := os.MkdirAll(p.snapsDir, 0o755); err != nil {
		return fmt.Errorf("create snapshots dir: %w", err)
	}
	if err := vm.takeSnapshot(snapFile, memFile); err != nil {
		return fmt.Errorf("take snapshot: %w", err)
	}

	p.mu.Lock()
	p.snapshots[lang] = SnapshotPair{SnapshotFile: snapFile, MemFile: memFile}
	p.mu.Unlock()

	log.Info("snapshot captured",
		zap.String("snapshot_file", snapFile),
		zap.String("mem_file", memFile),
	)
	return nil
}

// Get returns the snapshot pair for a language, or false if unavailable.
func (p *SnapshotPool) Get(lang string) (SnapshotPair, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	pair, ok := p.snapshots[lang]
	return pair, ok
}

// Languages returns all languages with a ready snapshot.
func (p *SnapshotPool) Languages() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	langs := make([]string, 0, len(p.snapshots))
	for l := range p.snapshots {
		langs = append(langs, l)
	}
	return langs
}
