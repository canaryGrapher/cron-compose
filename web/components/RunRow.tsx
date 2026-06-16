import Link from "next/link";
import type { CSSProperties } from "react";
import type { Run } from "@/lib/types";

const statusColor: Record<Run["status"], string> = {
  pending: "var(--muted)",
  running: "var(--info)",
  succeeded: "var(--ok)",
  failed: "var(--danger)",
  timed_out: "var(--danger)",
  canceled: "var(--muted)",
  skipped: "var(--muted)",
};

function fmtDuration(ms?: number) {
  if (ms === undefined) return "";
  if (ms < 1000) return `${ms} ms`;
  if (ms < 60_000) return `${(ms / 1000).toFixed(1)}s`;
  return `${Math.floor(ms / 60_000)}m ${Math.floor((ms % 60_000) / 1000)}s`;
}

export function RunRow({ run }: { run: Run }) {
  return (
    <Link href={`/runs/${run.id}`} className="panel" style={{ display: "block", color: "var(--text)" }}>
      <div className="row">
        <div>
          <div style={{ fontSize: 12 }} className="subtle">
            {new Date(run.created_at).toLocaleString()}
          </div>
          <div style={{ fontSize: 12, marginTop: 2 }}>
            trigger: {run.trigger}
            {run.duration_ms !== undefined && <> · {fmtDuration(run.duration_ms)}</>}
            {run.exit_code !== undefined && <> · exit {run.exit_code}</>}
          </div>
        </div>
        <span className="status" style={{ "--status-color": statusColor[run.status] } as CSSProperties}>
          {run.status}
        </span>
      </div>
    </Link>
  );
}
