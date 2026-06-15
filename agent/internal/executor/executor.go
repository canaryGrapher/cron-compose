// Package executor runs a job's script on the local machine with timeout, log capture,
// and optional resource limits (CPU + memory) via systemd-run.
package executor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"syscall"
	"time"
)

// LogSink receives chunked output as a job runs. seq increments per stream.
type LogSink func(stream string, seq int, data []byte)

// Job is the minimum the executor needs to run something.
type Job struct {
	Interpreter     string            // bash | sh | python3 | node
	ScriptBody      string
	Env             map[string]string
	WorkingDir      string
	RunAsUser       string // not yet honored in this skeleton
	TimeoutSeconds  int
	CPUQuotaPercent int // 0 = unlimited; 100 = one full core
	MemoryMaxMB     int // 0 = unlimited
}

// Result summarizes how a run ended.
type Result struct {
	Status     string // succeeded | failed | timed_out
	ExitCode   int
	DurationMs int
	Err        string
}

// Run executes the job, streams logs through sink, and returns the result.
func Run(ctx context.Context, j Job, sink LogSink) Result {
	start := time.Now()
	timeout := time.Duration(j.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Minute
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	interp := j.Interpreter
	if interp == "" {
		interp = "bash"
	}

	prog, args := buildCommand(interp, j)
	cmd := exec.CommandContext(runCtx, prog, args...)
	cmd.Dir = j.WorkingDir
	cmd.Env = mergeEnv(j.Env)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return Result{
			Status:     "failed",
			DurationMs: int(time.Since(start).Milliseconds()),
			Err:        err.Error(),
		}
	}

	go pumpLines(stdout, "stdout", sink)
	go pumpLines(stderr, "stderr", sink)

	err := cmd.Wait()
	dur := int(time.Since(start).Milliseconds())

	if runCtx.Err() == context.DeadlineExceeded {
		return Result{Status: "timed_out", DurationMs: dur, Err: "timeout exceeded"}
	}
	if err == nil {
		return Result{Status: "succeeded", ExitCode: 0, DurationMs: dur}
	}
	if ee, ok := err.(*exec.ExitError); ok {
		code := ee.ExitCode()
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			return Result{Status: "failed", ExitCode: -int(ws.Signal()), DurationMs: dur, Err: ws.Signal().String()}
		}
		return Result{Status: "failed", ExitCode: code, DurationMs: dur, Err: err.Error()}
	}
	return Result{Status: "failed", DurationMs: dur, Err: err.Error()}
}

// buildCommand returns (program, args) for cmd.Start. If the job declares any limits
// AND systemd-run is available on PATH, the script is wrapped under a transient scope
// so the kernel enforces the caps. Otherwise the interpreter runs directly.
func buildCommand(interp string, j Job) (string, []string) {
	if j.CPUQuotaPercent <= 0 && j.MemoryMaxMB <= 0 {
		return interp, []string{"-c", j.ScriptBody}
	}
	if _, err := exec.LookPath("systemd-run"); err != nil {
		// systemd-run unavailable: log via the result later by running unlimited.
		return interp, []string{"-c", j.ScriptBody}
	}
	args := []string{"--quiet", "--scope", "--collect"}
	if j.CPUQuotaPercent > 0 {
		args = append(args, "-p", fmt.Sprintf("CPUQuota=%d%%", j.CPUQuotaPercent))
	}
	if j.MemoryMaxMB > 0 {
		args = append(args, "-p", fmt.Sprintf("MemoryMax=%dM", j.MemoryMaxMB))
	}
	args = append(args, "--", interp, "-c", j.ScriptBody)
	return "systemd-run", args
}

func pumpLines(r io.Reader, stream string, sink LogSink) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	seq := 0
	for scanner.Scan() {
		sink(stream, seq, append([]byte{}, scanner.Bytes()...))
		seq++
	}
}

func mergeEnv(envMap map[string]string) []string {
	out := make([]string, 0, len(envMap))
	for k, v := range envMap {
		out = append(out, k+"="+v)
	}
	return out
}
