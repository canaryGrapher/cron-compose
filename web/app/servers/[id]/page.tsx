import Link from "next/link";
import { apiGet } from "@/lib/api";
import type { Job, ListResponse, Server } from "@/lib/types";
import { JobRow } from "@/components/JobRow";

type Props = { params: Promise<{ id: string }> };

export default async function ServerDetailPage({ params }: Props) {
  const { id } = await params;
  let server: Server | null = null;
  let jobs: Job[] = [];
  let error: string | null = null;
  try {
    server = await apiGet<Server>(`/servers/${id}`);
    const data = await apiGet<ListResponse<Job>>(`/jobs?server=${id}`);
    jobs = data.items;
  } catch (e) {
    error = (e as Error).message;
  }

  if (error || !server) {
    return (
      <div className="panel" style={{ color: "var(--danger)" }}>
        Could not load server: <code>{error ?? "not found"}</code>
      </div>
    );
  }

  return (
    <>
      <div className="row">
        <div>
          <h1>{server.name}</h1>
          <p className="subtle">
            {server.os || "unknown OS"} / {server.arch || "unknown arch"} · status: {server.status}
          </p>
        </div>
        <Link href={`/servers/${server.id}/jobs/new`} className="button">
          New job
        </Link>
      </div>

      <h2>Jobs</h2>
      {jobs.length === 0 ? (
        <div className="panel">
          <p className="subtle">No jobs yet on this server.</p>
        </div>
      ) : (
        <div className="stack">
          {jobs.map((j) => (
            <JobRow key={j.id} job={j} />
          ))}
        </div>
      )}
    </>
  );
}
