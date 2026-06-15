import Link from "next/link";
import { apiGet } from "@/lib/api";
import type { Job, ListResponse, Run } from "@/lib/types";
import { RunRow } from "@/components/RunRow";
import { RunNowButton } from "./RunNowButton";

type Props = { params: Promise<{ id: string }> };

export default async function JobDetailPage({ params }: Props) {
  const { id } = await params;
  let job: Job | null = null;
  let runs: Run[] = [];
  let error: string | null = null;
  try {
    job = await apiGet<Job>(`/jobs/${id}`);
    const data = await apiGet<ListResponse<Run>>(`/jobs/${id}/runs?limit=20`);
    runs = data.items;
  } catch (e) {
    error = (e as Error).message;
  }

  if (error || !job) {
    return (
      <div className="panel" style={{ color: "var(--danger)" }}>
        Could not load job: <code>{error ?? "not found"}</code>
      </div>
    );
  }

  return (
    <>
      <div className="row">
        <div>
          <Link href={`/servers/${job.server_id}`} style={{ fontSize: 12 }}>
            ← back to server
          </Link>
          <h1>{job.name}</h1>
          <p className="subtle">
            <code>{job.schedule_cron}</code> ({job.timezone}) · {job.interpreter} ·{" "}
            {job.enabled ? "enabled" : "disabled"} · v{job.current_version}
          </p>
        </div>
        <RunNowButton jobId={job.id} />
      </div>

      <h2>Script</h2>
      <pre
        className="panel"
        style={{
          whiteSpace: "pre-wrap",
          fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace",
          fontSize: 13,
        }}
      >
        {job.script_body}
      </pre>

      <h2>Recent runs</h2>
      {runs.length === 0 ? (
        <div className="panel">
          <p className="subtle">No runs yet.</p>
        </div>
      ) : (
        <div className="stack">
          {runs.map((r) => (
            <RunRow key={r.id} run={r} />
          ))}
        </div>
      )}
    </>
  );
}
