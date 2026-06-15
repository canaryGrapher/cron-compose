package runtime

import (
	"context"
	"crypto/rand"
	"time"

	"github.com/oklog/ulid/v2"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/croncompose/croncompose/agent/internal/executor"
	agentv1 "github.com/croncompose/croncompose/proto/agent/v1"
)

// onSchedulerFire is the callback the scheduler invokes when a job's cron triggers.
func (r *Runtime) onSchedulerFire(jobID string) {
	r.executeRun(context.Background(), jobID, newULID(), "schedule")
}

// executeRun runs one job execution: applies concurrency policy, emits RunStarted,
// streams LogChunks, emits RunFinished. Safe to call concurrently for different jobs.
func (r *Runtime) executeRun(ctx context.Context, jobID, runID, trigger string) {
	r.jobsMu.RLock()
	def, ok := r.jobs[jobID]
	r.jobsMu.RUnlock()
	if !ok {
		r.log.Warn("fire for unknown job", "job_id", jobID)
		return
	}

	if !r.acquireRunSlot(jobID, def.ConcurrencyPolicy) {
		// skip policy: another run already in progress. Could record a skipped run
		// upstream in Phase 2; for MVP we just log.
		r.log.Info("skipping run, already in progress", "job_id", jobID, "policy", def.ConcurrencyPolicy)
		return
	}
	runCtx, cancel := context.WithCancel(ctx)
	r.trackRun(runID, jobID, cancel)
	defer r.releaseRun(runID, jobID)

	startedAt := time.Now().UTC()
	r.queue(&agentv1.AgentMessage{
		Body: &agentv1.AgentMessage_RunStarted{RunStarted: &agentv1.RunStarted{
			RunId:        runID,
			JobId:        jobID,
			JobVersionId: def.VersionID,
			Trigger:      trigger,
			StartedAt:    timestamppb.New(startedAt),
		}},
	})

	mergedEnv := mergeEnv(def.Env, def.Secrets)
	job := executor.Job{
		Interpreter:     def.Interpreter,
		ScriptBody:      def.ScriptBody,
		Env:             mergedEnv,
		WorkingDir:      def.WorkingDir,
		RunAsUser:       def.RunAsUser,
		TimeoutSeconds:  def.TimeoutSeconds,
		CPUQuotaPercent: def.CPUQuotaPercent,
		MemoryMaxMB:     def.MemoryMaxMB,
	}
	sink := func(stream string, seq int, data []byte) {
		r.queue(&agentv1.AgentMessage{
			Body: &agentv1.AgentMessage_LogChunk{LogChunk: &agentv1.LogChunk{
				RunId:  runID,
				Stream: stream,
				Seq:    int32(seq),
				Data:   append([]byte(nil), data...),
			}},
		})
	}
	result := executor.Run(runCtx, job, sink)

	r.queue(&agentv1.AgentMessage{
		Body: &agentv1.AgentMessage_RunFinished{RunFinished: &agentv1.RunFinished{
			RunId:      runID,
			Status:     result.Status,
			ExitCode:   int32(result.ExitCode),
			FinishedAt: timestamppb.Now(),
			DurationMs: int32(result.DurationMs),
			Error:      result.Err,
		}},
	})
}

// acquireRunSlot enforces the concurrency policy for a job.
func (r *Runtime) acquireRunSlot(jobID, policy string) bool {
	r.runsMu.Lock()
	defer r.runsMu.Unlock()
	if r.jobBusy[jobID] > 0 {
		if policy == "allow" {
			return true
		}
		// "skip" (default) and "queue" both refuse a second concurrent start. Queue
		// semantics will need a per-job FIFO in Phase 2.
		return false
	}
	return true
}

func (r *Runtime) trackRun(runID, jobID string, cancel context.CancelFunc) {
	r.runsMu.Lock()
	defer r.runsMu.Unlock()
	r.runIndex[runID] = activeRun{jobID: jobID, cancel: cancel}
	r.jobBusy[jobID]++
}

func (r *Runtime) releaseRun(runID, jobID string) {
	r.runsMu.Lock()
	defer r.runsMu.Unlock()
	delete(r.runIndex, runID)
	if r.jobBusy[jobID] > 0 {
		r.jobBusy[jobID]--
	}
}

// cancelRun cancels the single in-progress run matching runID. The executor marks the
// run as canceled when its process exits.
func (r *Runtime) cancelRun(runID string) {
	r.runsMu.Lock()
	defer r.runsMu.Unlock()
	if a, ok := r.runIndex[runID]; ok {
		a.cancel()
	}
}

func mergeEnv(base, secrets map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(secrets))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range secrets {
		out[k] = v
	}
	return out
}

func newULID() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
}
