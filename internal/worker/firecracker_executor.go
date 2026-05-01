//go:build linux

package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/KhachikAstoyan/capstone/internal/controlplane/domain"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// FCConfig holds all paths and tunables for Firecracker/jailer execution.
type FCConfig struct {
	FirecrackerBin string // e.g. /usr/bin/firecracker
	JailerBin      string // e.g. /usr/bin/jailer
	KernelPath     string // e.g. /var/lib/fc/vmlinux
	ChrootBase     string // e.g. /srv/jailer  (jailer creates subdirs here)
	SnapshotsDir   string // e.g. /var/lib/fc/snapshots  (snapshot files live here)
	JailerUID      int    // UID jailer drops to (e.g. 900)
	JailerGID      int    // GID jailer drops to (e.g. 900)
	VCPU           int    // vCPUs per VM (default 1)
	MemMB          int    // VM memory in MiB (default 128)
}

func (c FCConfig) vcpu() int64 {
	if c.VCPU > 0 {
		return int64(c.VCPU)
	}
	return 1
}

func (c FCConfig) memMB() int64 {
	if c.MemMB > 0 {
		return int64(c.MemMB)
	}
	return 128
}

// fcJobRequest is the JSON payload sent to the in-VM fc-agent over vsock.
type fcJobRequest struct {
	SourceFile string   `json:"source_file"`
	SourceText string   `json:"source_text"`
	RunnerFile string   `json:"runner_file"`
	RunnerText string   `json:"runner_text"`
	RunCmd     []string `json:"run_cmd"`
	TestsJSON  string   `json:"tests_json"`
}

// FirecrackerExecutor executes jobs inside Firecracker microVMs.
//
// At construction time it boots one VM per language, captures a full
// memory+state snapshot, and kills the boot VMs. Each subsequent job
// restores a VM from that snapshot (~100–200 ms), sends the job to the
// in-VM fc-agent over vsock, reads the JSON result, then kills the VM.
type FirecrackerExecutor struct {
	pool      *SnapshotPool
	languages map[string]LangConfig
	cfg       FCConfig
	log       *zap.Logger
}

// NewFirecrackerExecutor boots snapshot VMs for all languages in rootfsByLang
// and returns an executor ready to serve jobs.
// rootfsByLang maps language key → host path to the ext4 rootfs image.
func NewFirecrackerExecutor(
	fc FCConfig,
	languages map[string]LangConfig,
	rootfsByLang map[string]string,
	log *zap.Logger,
) (*FirecrackerExecutor, error) {
	if log == nil {
		log = zap.NewNop()
	}
	pool := NewSnapshotPool(fc, fc.SnapshotsDir, log)
	log.Info("initializing firecracker snapshot pool",
		zap.Int("languages", len(rootfsByLang)),
	)
	if err := pool.Init(rootfsByLang); err != nil {
		return nil, fmt.Errorf("snapshot pool init: %w", err)
	}
	log.Info("firecracker executor ready",
		zap.Strings("languages", pool.Languages()),
	)
	return &FirecrackerExecutor{
		pool:      pool,
		languages: languages,
		cfg:       fc,
		log:       log,
	}, nil
}

// Execute implements Executor.
func (e *FirecrackerExecutor) Execute(ctx context.Context, a *domain.Assignment) (*ExecutionResult, error) {
	log := e.log.With(
		zap.String("job_id", a.JobID.String()),
		zap.String("language", a.Language),
		zap.Int("test_cases", len(a.TestCases)),
	)

	lang, ok := e.languages[a.Language]
	if !ok {
		log.Warn("unsupported language")
		return failResult(fmt.Sprintf("unsupported language: %s", a.Language)), nil
	}

	snap, ok := e.pool.Get(a.Language)
	if !ok {
		log.Error("no snapshot available for language")
		return failResult(fmt.Sprintf("no snapshot for language: %s", a.Language)), nil
	}

	vmID := "job-" + uuid.New().String()[:12]
	log = log.With(zap.String("vm_id", vmID))
	log.Info("restoring VM from snapshot")

	vm, err := restoreVM(e.cfg, vmID, snap.SnapshotFile, snap.MemFile, log)
	if err != nil {
		log.Error("restore VM failed", zap.Error(err))
		return failResult(fmt.Sprintf("restore VM: %v", err)), nil
	}
	defer func() {
		vm.kill()
		vm.cleanup()
		log.Info("VM killed and cleaned up")
	}()

	log.Info("waiting for agent after restore")
	if err := vm.waitForAgent(10 * time.Second); err != nil {
		log.Error("agent not ready after restore", zap.Error(err))
		return failResult(fmt.Sprintf("agent not ready: %v", err)), nil
	}

	// Build and send job request.
	source := ""
	if a.SourceText != nil {
		source = *a.SourceText
	}
	runnerText := fmt.Sprintf(lang.RunnerTemplate, a.TimeLimitMs, a.TimeLimitMs)

	tcBytes, err := json.Marshal(a.TestCases)
	if err != nil {
		return nil, fmt.Errorf("marshal test cases: %w", err)
	}

	req := fcJobRequest{
		SourceFile: lang.SourceFile,
		SourceText: source,
		RunnerFile: lang.RunnerFile,
		RunnerText: runnerText,
		RunCmd:     lang.RunCmd,
		TestsJSON:  string(tcBytes),
	}

	// Wall-clock budget: sum of per-test limits + compile overhead + margin.
	wallTimeout := time.Duration(len(a.TestCases)*a.TimeLimitMs+15_000) * time.Millisecond
	execCtx, cancel := context.WithTimeout(ctx, wallTimeout)
	defer cancel()

	log.Info("sending job to agent", zap.Duration("wall_timeout", wallTimeout))
	output, err := e.runJob(execCtx, vm, req)
	if err != nil {
		log.Error("job execution failed", zap.Error(err))
		return failResult(fmt.Sprintf("run job: %v", err)), nil
	}

	result, err := parseResults(a, output)
	if err != nil {
		log.Error("parse results failed", zap.Error(err))
		return failResult(fmt.Sprintf("parse results: %v", err)), nil
	}
	log.Info("job complete", zap.String("verdict", result.OverallVerdict))
	return result, nil
}

// runJob dials the agent, sends the request, and reads the runner output.
func (e *FirecrackerExecutor) runJob(ctx context.Context, vm *fcVM, req fcJobRequest) (string, error) {
	conn, err := vm.dialVsock()
	if err != nil {
		return "", fmt.Errorf("dial vsock: %w", err)
	}
	defer conn.Close()

	// Propagate context deadline to the connection.
	if dl, ok := ctx.Deadline(); ok {
		conn.SetDeadline(dl) //nolint:errcheck
	}

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return "", fmt.Errorf("send job request: %w", err)
	}

	// Read all output (agent closes connection after writing result).
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 4096)
	for {
		n, readErr := conn.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if readErr != nil {
			break // EOF or deadline — either way we have what we need
		}
	}
	if len(buf) == 0 {
		return "", fmt.Errorf("agent returned empty response")
	}
	return string(buf), nil
}

// parseResults is shared with DockerExecutor (same runner output format).
func parseResults(a *domain.Assignment, output string) (*ExecutionResult, error) {
	// Reuse the DockerExecutor's parseResults logic by constructing a temp
	// DockerExecutor — or just inline the identical logic here.
	type runnerResult struct {
		ID      string `json:"id"`
		Verdict string `json:"verdict"`
		TimeMs  int    `json:"time_ms"`
		Actual  string `json:"actual"`
		Stdout  string `json:"stdout"`
		Stderr  string `json:"stderr"`
	}

	var raw []runnerResult
	if err := json.Unmarshal([]byte(trimToJSON(output)), &raw); err != nil {
		return failResult(fmt.Sprintf("output parse error: %v — output: %.200s", err, output)), nil
	}

	var (
		tcResults      []domain.TestcaseResultInput
		totalTimeMs    int
		overallVerdict = "Accepted"
		compilerOutput *string
	)

	for _, r := range raw {
		if r.Verdict != "Accepted" && overallVerdict == "Accepted" {
			overallVerdict = r.Verdict
			if r.Verdict == "CompilationError" && r.Stderr != "" {
				s := r.Stderr
				compilerOutput = &s
			}
		}
		totalTimeMs += r.TimeMs
		timeMs := r.TimeMs
		actual := r.Actual
		stdout := r.Stdout
		tcResults = append(tcResults, domain.TestcaseResultInput{
			TestcaseID:   r.ID,
			Verdict:      r.Verdict,
			TimeMs:       &timeMs,
			ActualOutput: &actual,
			StdoutRef:    &stdout,
		})
	}

	return &ExecutionResult{
		OverallVerdict:  overallVerdict,
		TotalTimeMs:     &totalTimeMs,
		CompilerOutput:  compilerOutput,
		TestcaseResults: tcResults,
	}, nil
}

// trimToJSON finds the first '[' in s, which is where the runner JSON starts.
func trimToJSON(s string) string {
	for i, c := range s {
		if c == '[' {
			return s[i:]
		}
	}
	return s
}
