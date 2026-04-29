package worker

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/KhachikAstoyan/capstone/internal/controlplane/domain"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	dockerclient "github.com/docker/docker/client"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Language configuration
// ---------------------------------------------------------------------------

// LangConfig describes how to compile and run code for a given language inside
// the Docker container.
type LangConfig struct {
	// Docker image to use.  Must already be present on the host (workers
	// should pre-pull images at startup).
	Image string

	// SourceFile is the name written inside /workspace (e.g. "solution.py").
	SourceFile string

	// CompileCmd is run before execution for compiled languages.
	// Empty slice means the language is interpreted — skip compilation.
	CompileCmd []string

	// RunCmd is the command that runs one test case.
	// The runner script will be invoked instead when RunnerFile is set.
	RunCmd []string

	// RunnerFile is the name of the generated test runner placed in /workspace.
	RunnerFile string

	// RunnerTemplate is a Go text/template that generates the runner script.
	// Available variables: .SourceFile, .TimeLimitMs
	RunnerTemplate string

	// PidsLimit overrides the default container PID limit (64).
	// Compiled languages like Go need a higher value because the toolchain
	// forks multiple processes during compilation.
	PidsLimit int64

	// TmpfsOpts overrides the default /tmp tmpfs mount options ("size=64m").
	// Compiled languages like Go write build artifacts and execute binaries
	// from /tmp, so they need more space and exec permission.
	TmpfsOpts string
}

// DefaultLanguages is the set of language configs bundled with the worker.
// Add more entries here as new language images are integrated.
var DefaultLanguages = map[string]LangConfig{
	"python": {
		Image:          "capstone-python-runner:latest",
		SourceFile:     "solution.py",
		RunnerFile:     "runner.py",
		RunnerTemplate: pythonRunnerTemplate,
		RunCmd:         []string{"python3", "/workspace/runner.py"},
	},
	"javascript": {
		Image:          "capstone-js-runner:latest",
		SourceFile:     "solution.js",
		RunnerFile:     "runner.js",
		RunnerTemplate: jsRunnerTemplate,
		RunCmd:         []string{"node", "/workspace/runner.js"},
	},
	"go": {
		Image:          "capstone-go-runner:latest",
		SourceFile:     "solution.go",
		RunnerFile:     "runner_go.py",
		RunnerTemplate: goRunnerTemplate,
		RunCmd:         []string{"python3", "/workspace/runner_go.py"},
		PidsLimit: 256,
		TmpfsOpts: "size=512m,exec",
	},
	"java": {
		Image:          "capstone-java-runner:latest",
		SourceFile:     "Main.java",
		RunnerFile:     "runner_java.py",
		RunnerTemplate: javaRunnerTemplate,
		RunCmd:         []string{"python3", "/workspace/runner_java.py"},
		PidsLimit: 256,
		TmpfsOpts: "size=512m,exec",
	},
}

// ---------------------------------------------------------------------------
// Runner templates (embedded from runners/)
// ---------------------------------------------------------------------------

//go:embed runners/python.py
var pythonRunnerTemplate string

//go:embed runners/javascript.js
var jsRunnerTemplate string

//go:embed runners/go.py
var goRunnerTemplate string

//go:embed runners/java.py
var javaRunnerTemplate string

// ---------------------------------------------------------------------------
// DockerExecutor
// ---------------------------------------------------------------------------

// DockerExecutor runs user code inside a Docker container for each job.
//
// Execution model (stdin/stdout):
//  1. A temp directory is created on the host with the user's source file,
//     a generated language-specific runner, and a tests.json file.
//  2. The directory is bind-mounted into the container at /workspace (read-only
//     for the source; the runner executes each test case as a subprocess).
//  3. The container runs the runner, which outputs a JSON array of results.
//  4. The worker parses the JSON and maps results onto domain types.
//  5. The temp directory and container are always removed.
//
// Security hardening applied to every container:
//   - No network access (NetworkMode: none)
//   - Read-only root filesystem
//   - Memory limit (from job's memory_limit_mb)
//   - PID limit (prevents fork bombs)
//   - No new privileges
//   - Dropped all Linux capabilities
type DockerExecutor struct {
	client    *dockerclient.Client
	languages map[string]LangConfig
	log       *zap.Logger
}

// NewDockerExecutor creates a DockerExecutor using the host Docker daemon.
func NewDockerExecutor(languages map[string]LangConfig, log *zap.Logger) (*DockerExecutor, error) {
	if log == nil {
		log = zap.NewNop()
	}
	log.Info("initializing docker executor")
	cli, err := dockerclient.NewClientWithOpts(
		dockerclient.FromEnv,
		dockerclient.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to docker daemon: %w", err)
	}
	log.Info("docker client initialized", zap.Int("configured_languages", len(languages)))
	return &DockerExecutor{client: cli, languages: languages, log: log}, nil
}

// Execute implements Executor.
func (e *DockerExecutor) Execute(ctx context.Context, a *domain.Assignment) (*ExecutionResult, error) {
	log := e.log.With(
		zap.String("job_id", a.JobID.String()),
		zap.String("submission_id", a.SubmissionID.String()),
		zap.String("language", a.Language),
		zap.Int("test_cases", len(a.TestCases)),
		zap.Int("time_limit_ms", a.TimeLimitMs),
		zap.Int("memory_limit_mb", a.MemoryLimitMb),
	)
	log.Info("docker executor received job")

	lang, ok := e.languages[a.Language]
	if !ok {
		log.Warn("skipping docker container because language is unsupported")
		return failResult(fmt.Sprintf("unsupported language: %s", a.Language)), nil
	}
	if len(a.TestCases) == 0 {
		log.Warn("skipping docker container because assignment has no test cases")
		return failResult("no test cases provided"), nil
	}
	log.Info("resolved language runtime",
		zap.String("image", lang.Image),
		zap.String("source_file", lang.SourceFile),
		zap.String("runner_file", lang.RunnerFile),
		zap.Strings("run_cmd", lang.RunCmd),
	)

	// 1. Build workspace on the host.
	log.Info("building execution workspace")
	workDir, err := e.buildWorkspace(a, lang)
	if err != nil {
		log.Error("failed to build execution workspace", zap.Error(err))
		return nil, fmt.Errorf("build workspace: %w", err)
	}
	log.Info("execution workspace ready", zap.String("work_dir", workDir))
	defer func() {
		log.Info("removing execution workspace", zap.String("work_dir", workDir))
		if err := os.RemoveAll(workDir); err != nil {
			log.Warn("failed to remove execution workspace", zap.String("work_dir", workDir), zap.Error(err))
		}
	}()

	// 2. Run the container.
	log.Info("starting docker container execution")
	output, runErr := e.runContainer(ctx, a, lang, workDir)
	if runErr != nil {
		log.Error("docker container execution failed", zap.Error(runErr))
		return failResult(fmt.Sprintf("container error: %v", runErr)), nil
	}
	log.Info("docker container execution returned output", zap.Int("output_bytes", len(output)))

	// 3. Parse runner output.
	log.Info("parsing runner output")
	result, err := e.parseResults(a, output)
	if err != nil {
		log.Error("runner output parsing failed", zap.Error(err))
		return nil, err
	}
	log.Info("runner output parsed",
		zap.String("overall_verdict", result.OverallVerdict),
		zap.Int("testcase_results", len(result.TestcaseResults)),
	)
	return result, nil
}

// ---------------------------------------------------------------------------
// buildWorkspace
// ---------------------------------------------------------------------------

func (e *DockerExecutor) buildWorkspace(a *domain.Assignment, lang LangConfig) (string, error) {
	dir, err := os.MkdirTemp("", "capstone-job-*")
	if err != nil {
		return "", err
	}

	// Write user's source code.
	source := ""
	if a.SourceText != nil {
		source = *a.SourceText
	}
	if err := os.WriteFile(filepath.Join(dir, lang.SourceFile), []byte(source), 0o644); err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("write source: %w", err)
	}

	// Write test cases as JSON.
	tcJSON, err := json.Marshal(a.TestCases)
	if err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("marshal test cases: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tests.json"), tcJSON, 0o644); err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("write tests.json: %w", err)
	}

	// Write the generated runner script.
	runner := fmt.Sprintf(lang.RunnerTemplate, a.TimeLimitMs, a.TimeLimitMs)
	if err := os.WriteFile(filepath.Join(dir, lang.RunnerFile), []byte(runner), 0o644); err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("write runner: %w", err)
	}

	return dir, nil
}

// ---------------------------------------------------------------------------
// runContainer
// ---------------------------------------------------------------------------

func (e *DockerExecutor) runContainer(ctx context.Context, a *domain.Assignment, lang LangConfig, workDir string) (string, error) {
	log := e.log.With(
		zap.String("job_id", a.JobID.String()),
		zap.String("submission_id", a.SubmissionID.String()),
		zap.String("language", a.Language),
		zap.String("image", lang.Image),
		zap.String("work_dir", workDir),
	)
	memBytes := int64(a.MemoryLimitMb) * 1024 * 1024

	// Total wall-clock budget: sum of all per-test limits plus a small overhead.
	wallTimeout := time.Duration(len(a.TestCases)*a.TimeLimitMs+5000) * time.Millisecond
	log.Info("computed docker execution limits",
		zap.Int64("memory_bytes", memBytes),
		zap.Duration("wall_timeout", wallTimeout),
	)
	runCtx, cancel := context.WithTimeout(ctx, wallTimeout)
	defer cancel()

	cfg := &container.Config{
		Image:           lang.Image,
		Cmd:             lang.RunCmd,
		NetworkDisabled: true,
		WorkingDir:      "/workspace",
	}

	hostCfg := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:     mount.TypeBind,
				Source:   workDir,
				Target:   "/workspace",
				ReadOnly: true,
			},
		},
		Resources: container.Resources{
			Memory:     memBytes,
			MemorySwap: memBytes, // no swap
			PidsLimit:  int64ptr(pidsLimit(lang)),
		},
		SecurityOpt:    []string{"no-new-privileges"},
		ReadonlyRootfs: true,
		// Tmpfs lets the runner write to /tmp inside the read-only container.
		Tmpfs: map[string]string{"/tmp": tmpfsOpts(lang)},
		CapDrop:     []string{"ALL"},
		NetworkMode: "none",
	}

	log.Info("creating docker container",
		zap.Strings("cmd", lang.RunCmd),
		zap.Bool("network_disabled", cfg.NetworkDisabled),
		zap.Bool("readonly_rootfs", hostCfg.ReadonlyRootfs),
	)
	resp, err := e.client.ContainerCreate(runCtx, cfg, hostCfg, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("create container: %w", err)
	}
	containerID := resp.ID
	log.Info("docker container created", zap.String("container_id", containerID))
	// defer func() {
	// 	log.Info("removing docker container", zap.String("container_id", containerID))
	// 	if err := e.client.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true}); err != nil {
	// 		log.Warn("failed to remove docker container", zap.String("container_id", containerID), zap.Error(err))
	// 	}
	// }()

	log.Info("starting docker container", zap.String("container_id", containerID))
	if err := e.client.ContainerStart(runCtx, containerID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("start container: %w", err)
	}
	log.Info("docker container started", zap.String("container_id", containerID))

	// Wait for the container to finish.
	log.Info("waiting for docker container to finish", zap.String("container_id", containerID))
	statusCh, errCh := e.client.ContainerWait(runCtx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return "", fmt.Errorf("container wait: %w", err)
		}
	case status := <-statusCh:
		errorMessage := ""
		if status.Error != nil {
			errorMessage = status.Error.Message
		}
		log.Info("docker container finished",
			zap.String("container_id", containerID),
			zap.Int64("status_code", status.StatusCode),
			zap.String("error_message", errorMessage),
		)
	case <-runCtx.Done():
		return "", fmt.Errorf("job timed out after %v", wallTimeout)
	}

	// Collect stdout (the runner prints JSON here).
	log.Info("collecting docker container stdout", zap.String("container_id", containerID))
	logs, err := e.client.ContainerLogs(context.Background(), containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: false,
	})
	if err != nil {
		return "", fmt.Errorf("get container logs: %w", err)
	}
	defer logs.Close()

	// Docker multiplexes stdout/stderr with an 8-byte header per frame.
	// io.ReadAll works fine because ContainerLogs with ShowStderr=false only
	// returns stdout frames; the header bytes are present but json.Unmarshal
	// skips leading non-JSON bytes gracefully — we strip them explicitly.
	raw, err := io.ReadAll(logs)
	if err != nil {
		return "", fmt.Errorf("read logs: %w", err)
	}
	log.Info("docker container stdout collected",
		zap.String("container_id", containerID),
		zap.Int("raw_bytes", len(raw)),
	)

	// Strip Docker's multiplexing headers (each frame: 8 bytes header + payload).
	return stripDockerHeaders(raw), nil
}

// ---------------------------------------------------------------------------
// parseResults
// ---------------------------------------------------------------------------

type runnerResult struct {
	ID      string `json:"id"`
	Verdict string `json:"verdict"`
	TimeMs  int    `json:"time_ms"`
	Actual  string `json:"actual"`
	Stdout  string `json:"stdout"`
	Stderr  string `json:"stderr"`
}

func (e *DockerExecutor) parseResults(_ *domain.Assignment, output string) (*ExecutionResult, error) {
	var raw []runnerResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &raw); err != nil {
		return failResult(fmt.Sprintf("runner output parse error: %v — output: %.200s", err, output)), nil
	}

	var (
		tcResults      []domain.TestcaseResultInput
		totalTimeMs    int
		allAccepted    = true
		overallVerdict = "Accepted"
		compilerOutput *string
	)

	for _, r := range raw {
		if r.Verdict != "Accepted" {
			allAccepted = false
			if overallVerdict == "Accepted" {
				overallVerdict = r.Verdict
				if r.Verdict == "CompilationError" && r.Stderr != "" {
					s := r.Stderr
					compilerOutput = &s
				}
			}
		}
		totalTimeMs += r.TimeMs
		timeMs := r.TimeMs
		actual := r.Actual
		stdout := r.Stdout
		entry := domain.TestcaseResultInput{
			TestcaseID:   r.ID,
			Verdict:      r.Verdict,
			TimeMs:       &timeMs,
			ActualOutput: &actual,
			StdoutRef:    &stdout,
		}
		tcResults = append(tcResults, entry)
	}
	_ = allAccepted

	return &ExecutionResult{
		OverallVerdict:  overallVerdict,
		TotalTimeMs:     &totalTimeMs,
		CompilerOutput:  compilerOutput,
		TestcaseResults: tcResults,
	}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func failResult(reason string) *ExecutionResult {
	return &ExecutionResult{OverallVerdict: "InternalError", CompilerOutput: &reason}
}

func int64ptr(v int64) *int64 { return &v }

func pidsLimit(lang LangConfig) int64 {
	if lang.PidsLimit > 0 {
		return lang.PidsLimit
	}
	return 64
}

func tmpfsOpts(lang LangConfig) string {
	if lang.TmpfsOpts != "" {
		return lang.TmpfsOpts
	}
	return "size=64m"
}

// stripDockerHeaders removes the 8-byte multiplexing header Docker prepends to
// each log frame, leaving only the raw payload bytes.
func stripDockerHeaders(b []byte) string {
	var sb strings.Builder
	for len(b) >= 8 {
		frameSize := int(b[4])<<24 | int(b[5])<<16 | int(b[6])<<8 | int(b[7])
		b = b[8:]
		if frameSize > len(b) {
			frameSize = len(b)
		}
		sb.Write(b[:frameSize])
		b = b[frameSize:]
	}
	return sb.String()
}
