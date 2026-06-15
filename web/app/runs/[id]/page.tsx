"use client";

import { useEffect, useState, use } from "react";
import type { LogLine, Run } from "@/lib/types";

type Props = { params: Promise<{ id: string }> };

const statusColor: Record<Run["status"], string> = {
  pending: "var(--muted)",
  running: "var(--accent)",
  succeeded: "var(--ok)",
  failed: "var(--danger)",
  timed_out: "var(--danger)",
  canceled: "var(--muted)",
  skipped: "var(--muted)",
};

export default function RunDetailPage({ params }: Props) {
  const { id } = use(params);
  const [run, setRun] = useState<Run | null>(null);
  const [logs, setLogs] = useState<LogLine[]>([]);
  const [error, setError] = useState<string | null>(null);

  // Initial metadata fetch.
  useEffect(() => {
    let cancelled = false;
    fetch(`/api/runs/${id}`)
      .then((r) => r.json() as Promise<Run>)
      .then((r) => {
        if (!cancelled) setRun(r);
      })
      .catch((e) => {
        if (!cancelled) setError((e as Error).message);
      });
    return () => {
      cancelled = true;
    };
  }, [id]);

  // Live log stream via SSE. The "done" event carries the final status.
  useEffect(() => {
    const es = new EventSource(`/api/runs/${id}/logs/stream`);
    es.addEventListener("log", (ev) => {
      const m = (ev as MessageEvent).data;
      try {
        const line = JSON.parse(m) as LogLine;
        setLogs((prev) => [...prev, line]);
      } catch {
        /* ignore malformed */
      }
    });
    es.addEventListener("done", (ev) => {
      const m = (ev as MessageEvent).data;
      try {
        const data = JSON.parse(m) as { status: Run["status"]; exit_code?: number };
        setRun((prev) => (prev ? { ...prev, status: data.status, exit_code: data.exit_code } : prev));
      } catch {
        /* ignore */
      }
      es.close();
    });
    es.onerror = () => {
      // Browser auto-reconnects unless we close. For a terminal run there's nothing to
      // recover, so just close on persistent error.
      es.close();
    };
    return () => es.close();
  }, [id]);

  if (error && !run) {
    return (
      <div className="panel" style={{ color: "var(--danger)" }}>
        Could not load run: <code>{error}</code>
      </div>
    );
  }
  if (!run) {
    return <p className="subtle">Loading...</p>;
  }

  return (
    <>
      <div className="row">
        <div>
          <h1>Run {run.id.slice(0, 8)}</h1>
          <p className="subtle">
            trigger: {run.trigger}
            {run.exit_code !== undefined && <> · exit {run.exit_code}</>}
            {run.duration_ms !== undefined && <> · {run.duration_ms} ms</>}
          </p>
        </div>
        <span style={{ color: statusColor[run.status], fontSize: 14, textTransform: "uppercase" }}>
          {run.status}
        </span>
      </div>

      <h2>Logs (live)</h2>
      <pre
        className="panel"
        style={{
          whiteSpace: "pre-wrap",
          fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace",
          fontSize: 12,
          maxHeight: 480,
          overflow: "auto",
        }}
      >
        {logs.length === 0 ? <span className="subtle">(no output yet)</span> : (
          logs.map((l) => (
            <div key={`${l.stream}-${l.seq}`}>
              <span style={{ color: l.stream === "stderr" ? "var(--danger)" : "var(--muted)" }}>
                [{l.stream}]
              </span>{" "}
              {l.chunk}
            </div>
          ))
        )}
      </pre>
    </>
  );
}
