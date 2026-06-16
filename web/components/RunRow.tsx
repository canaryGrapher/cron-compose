import Link from "next/link";
import type { Run } from "@/lib/types";
import { IconPlay } from "./icons";

const tone: Record<Run["status"], string> = {
  pending: "neutral",
  running: "info",
  succeeded: "ok",
  failed: "danger",
  timed_out: "danger",
  canceled: "neutral",
  skipped: "neutral",
};

function fmtDuration(ms?: number) {
  if (ms === undefined) return "";
  if (ms < 1000) return `${ms} ms`;
  if (ms < 60_000) return `${(ms / 1000).toFixed(1)}s`;
  return `${Math.floor(ms / 60_000)}m ${Math.floor((ms % 60_000) / 1000)}s`;
}

export function RunRow({ run }: { run: Run }) {
  return (
    <Link href={`/runs/${run.id}`} className="panel">
      <div className="row" style={{ alignItems: "flex-start" }}>
        <div className="cluster" style={{ flexWrap: "nowrap" }}>
          <span className="mini-icon"><IconPlay /></span>
          <div>
            <div style={{ fontWeight: 600, color: "var(--text)", fontSize: 13 }}>
              {new Date(run.created_at).toLocaleString()}
            </div>
            <div className="subtle" style={{ fontSize: 12 }}>
              {run.trigger}
              {run.duration_ms !== undefined && <> · {fmtDuration(run.duration_ms)}</>}
              {run.exit_code !== undefined && <> · exit {run.exit_code}</>}
            </div>
          </div>
        </div>
        <span className={`status ${tone[run.status]}`}>{run.status}</span>
      </div>
    </Link>
  );
}
