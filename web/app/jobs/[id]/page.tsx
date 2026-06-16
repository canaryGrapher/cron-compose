import Link from "next/link";
import { apiGet } from "@/lib/api";
import type { Job, ListResponse, Run } from "@/lib/types";
import { RunRow } from "@/components/RunRow";
import { RunNowButton } from "./RunNowButton";
import { IconChevronLeft } from "@/components/icons";

type Props = { params: Promise<{ id: string }> };

export default async function JobDetailPage({ params }: Props) {
  const { id } = await params;
  let job: Job | null = null;
  let runs: Run[] = [];
  let error: string | null = null;
  try {
    job = await apiGet<Job>(`/jobs/${id}`);
    runs = (await apiGet<ListResponse<Run>>(`/jobs/${id}/runs?limit=20`)).items;
  } catch (e) {
    error = (e as Error).message;
  }

  if (error || !job) {
    return <div className="form-error">Could not load job: <code>{error ?? "not found"}</code></div>;
  }

  const backHref = job.server_id ? `/servers/${job.server_id}` : "/jobs";

  return (
    <>
      <Link href={backHref} className="back-link"><IconChevronLeft /> Back</Link>
      <div className="page-head">
        <div>
          <h1>{job.name}</h1>
          <div className="cluster" style={{ marginTop: 6 }}>
            <span className={`status ${job.enabled ? "ok" : "neutral"}`}>{job.enabled ? "enabled" : "disabled"}</span>
            <span className="pill"><code>{job.schedule_cron}</code></span>
            <span className="pill">{job.timezone}</span>
            <span className="pill">{job.interpreter}</span>
            <span className="pill">v{job.current_version}</span>
          </div>
        </div>
        <div className="page-head-actions"><RunNowButton jobId={job.id} /></div>
      </div>

      {job.description && <p className="subtle" style={{ marginTop: -6, marginBottom: 18 }}>{job.description}</p>}

      <h2>Script</h2>
      <pre className="review-script" style={{ maxHeight: 420 }}>{job.script_body}</pre>

      <h2>Recent runs</h2>
      {runs.length === 0 ? (
        <div className="panel"><div className="empty">No runs yet.</div></div>
      ) : (
        <div className="stack">{runs.map((r) => <RunRow key={r.id} run={r} />)}</div>
      )}
    </>
  );
}
