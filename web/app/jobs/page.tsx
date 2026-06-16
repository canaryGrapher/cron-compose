import Link from "next/link";
import { apiGet } from "@/lib/api";
import type { Job, ListResponse, Server } from "@/lib/types";
import { JobRow } from "@/components/JobRow";
import { IconPlus } from "@/components/icons";

export default async function JobsPage() {
  let jobs: Job[] = [];
  let servers: Server[] = [];
  let error: string | null = null;
  try {
    const [jobsData, serversData] = await Promise.all([
      apiGet<ListResponse<Job>>("/jobs"),
      apiGet<ListResponse<Server>>("/servers"),
    ]);
    jobs = jobsData.items;
    servers = serversData.items;
  } catch (e) {
    error = (e as Error).message;
  }

  const newJobHref = servers.length > 0 ? `/servers/${servers[0].id}/jobs/new` : "/servers/new";

  return (
    <>
      <div className="page-head">
        <div>
          <h1>Jobs</h1>
          <p className="subtle">Every scheduled job across your fleet.</p>
        </div>
        <div className="page-head-actions">
          <Link href={newJobHref} className="button"><IconPlus /> New job</Link>
        </div>
      </div>

      {error && (
        <div className="form-error">Could not load jobs: <code>{error}</code></div>
      )}

      {!error && jobs.length === 0 && (
        <div className="panel">
          <div className="empty">
            No jobs yet.{" "}
            {servers.length > 0
              ? <Link href={newJobHref}>Create your first job</Link>
              : <Link href="/servers/new">Add a server</Link>} to get started.
          </div>
        </div>
      )}

      <div className="stack">
        {jobs.map((j) => <JobRow key={j.id} job={j} />)}
      </div>
    </>
  );
}
