import Link from "next/link";
import type { Job } from "@/lib/types";
import { IconJobs } from "./icons";

function targetSummary(job: Job): string {
  if (job.target_kind === "labels") {
    const pairs = Object.entries(job.target_labels).map(([k, v]) => `${k}=${v}`).join(", ");
    return pairs ? `labels: ${pairs}` : "label selector";
  }
  return "single server";
}

export function JobRow({ job }: { job: Job }) {
  return (
    <Link href={`/jobs/${job.id}`} className="panel">
      <div className="row" style={{ alignItems: "flex-start" }}>
        <div className="cluster" style={{ flexWrap: "nowrap", minWidth: 0 }}>
          <span className="mini-icon"><IconJobs /></span>
          <div style={{ minWidth: 0 }}>
            <div style={{ fontWeight: 700, fontSize: 15, color: "var(--text)" }}>{job.name}</div>
            <div className="subtle" style={{ fontSize: 12 }}>
              <code>{job.schedule_cron}</code> ({job.timezone}) · {job.interpreter} · v{job.current_version} · {targetSummary(job)}
            </div>
          </div>
        </div>
        <span className={`status ${job.enabled ? "ok" : "neutral"}`}>{job.enabled ? "enabled" : "disabled"}</span>
      </div>
    </Link>
  );
}
