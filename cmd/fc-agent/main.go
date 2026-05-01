//go:build linux

// cmd/fc-agent runs inside a Firecracker microVM as PID 1 (or a child of init).
//
// It listens on vsock port 52000 for job requests from the host worker,
// writes the workspace files, executes the runner, and returns JSON results.
//
// Build (cross-compile for VM):
//
//	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o agent ./cmd/fc-agent
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"golang.org/x/sys/unix"
)

const vsockPort = 52000

// JobRequest is sent from the host worker to the agent.
type JobRequest struct {
	// Ping == true means health-check only; agent responds with empty array.
	Ping bool `json:"ping,omitempty"`

	SourceFile string   `json:"source_file"`
	SourceText string   `json:"source_text"`
	RunnerFile string   `json:"runner_file"`
	RunnerText string   `json:"runner_text"`
	RunCmd     []string `json:"run_cmd"`
	TestsJSON  string   `json:"tests_json"`
}

func main() {
	fd, err := unix.Socket(unix.AF_VSOCK, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		fatalf("socket: %v", err)
	}
	sa := &unix.SockaddrVM{CID: unix.VMADDR_CID_ANY, Port: vsockPort}
	if err := unix.Bind(fd, sa); err != nil {
		fatalf("bind vsock port %d: %v", vsockPort, err)
	}
	if err := unix.Listen(fd, 4); err != nil {
		fatalf("listen: %v", err)
	}
	fmt.Fprintf(os.Stderr, "fc-agent ready on vsock port %d\n", vsockPort)

	for {
		connFd, _, err := unix.Accept(fd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "accept: %v\n", err)
			continue
		}
		// Sequential: one job at a time per VM.
		handleConn(connFd)
	}
}

func handleConn(fd int) {
	f := os.NewFile(uintptr(fd), "vsock-conn")
	defer f.Close()

	var req JobRequest
	if err := json.NewDecoder(f).Decode(&req); err != nil {
		fmt.Fprintf(os.Stderr, "decode request: %v\n", err)
		return
	}

	// Ping: just return empty array so the host knows the agent is alive.
	if req.Ping {
		f.WriteString("[]\n")
		return
	}

	if err := prepareWorkspace(req); err != nil {
		writeInternalError(f, fmt.Sprintf("workspace setup: %v", err))
		return
	}

	if len(req.RunCmd) == 0 {
		writeInternalError(f, "empty run_cmd")
		return
	}

	cmd := exec.Command(req.RunCmd[0], req.RunCmd[1:]...)
	out, err := cmd.Output()
	if err != nil {
		// Runner exited non-zero: return its stderr as InternalError.
		var exitErr *exec.ExitError
		stderr := ""
		if ok := asExitError(err, &exitErr); ok {
			stderr = string(exitErr.Stderr)
		}
		writeInternalError(f, fmt.Sprintf("runner: %v — %s", err, stderr))
		return
	}

	f.Write(out)
}

func prepareWorkspace(req JobRequest) error {
	if err := os.MkdirAll("/workspace", 0o755); err != nil {
		return err
	}
	if err := os.WriteFile("/workspace/"+req.SourceFile, []byte(req.SourceText), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile("/workspace/"+req.RunnerFile, []byte(req.RunnerText), 0o644); err != nil {
		return err
	}
	return os.WriteFile("/workspace/tests.json", []byte(req.TestsJSON), 0o644)
}

func writeInternalError(f *os.File, msg string) {
	safe, _ := json.Marshal(msg)
	fmt.Fprintf(f, `[{"id":"internal","verdict":"InternalError","time_ms":0,"actual":"","stdout":"","stderr":%s}]`+"\n", safe)
}

func asExitError(err error, out **exec.ExitError) bool {
	e, ok := err.(*exec.ExitError)
	if ok {
		*out = e
	}
	return ok
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "fc-agent: "+format+"\n", args...)
	os.Exit(1)
}
