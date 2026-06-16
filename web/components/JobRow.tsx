import Link from "next/link";
import type { CSSProperties } from "react";
import type { Job } from "@/lib/types";

function targetSummary(job: Job): string {
  if (job.target_kind === "labels") {
    const pairs = Object.entries(job.target_labels)
      .map(([k, v]) => `${k}=${v}`)
      .join(",");
    return `labels: ${pairs || "(none)"}`;
  }
  return "single server";
}

export function JobRow({ job }: { job: Job }) {
  return (
    <Link
      href={`/jobs/${job.id}`}
      className="panel"
      style={{ display: "block", color: "var(--text)" }}
    >
      <div className="row">
        <div>
          <div style={{ fontWeight: 700, fontSize: 15 }}>{job.name}</div>
          <div className="subtle" style={{ fontSize: 12, marginTop: 2 }}>
            <code>{job.schedule_cron}</code> ({job.timezone}) · {job.interpreter} · v
            {job.current_version} · {targetSummary(job)}
          </div>
        </div>
        <span
          className="status"
          style={{ "--status-color": job.enabled ? "var(--ok)" : "var(--muted)" } as CSSProperties}
        >
          {job.enabled ? "enabled" : "disabled"}
        </span>
      </div>
    </Link>
  );
}
