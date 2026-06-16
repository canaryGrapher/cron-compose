import { apiGet } from "@/lib/api";
import type { ListResponse } from "@/lib/types";

type AuditEntry = {
  id: number;
  actor_user_id?: string | null;
  action: string;
  target_type?: string | null;
  target_id?: string | null;
  metadata: Record<string, unknown>;
  ts: string;
};

export default async function AuditPage() {
  let entries: AuditEntry[] = [];
  let error: string | null = null;
  try {
    entries = (await apiGet<ListResponse<AuditEntry>>("/audit?limit=100")).items;
  } catch (e) {
    error = (e as Error).message;
  }

  return (
    <>
      <div className="page-head">
        <div>
          <h1>Audit log</h1>
          <p className="subtle">Most recent 100 entries. Admin only.</p>
        </div>
      </div>

      {error && <div className="form-error">Could not load: <code>{error}</code></div>}

      {!error && (
        <div className="panel" style={{ padding: 0 }}>
          <div className="table-wrap">
            <table className="data">
              <thead>
                <tr>
                  <th>When</th><th>Actor</th><th>Action</th><th>Target</th><th>Metadata</th>
                </tr>
              </thead>
              <tbody>
                {entries.length === 0 ? (
                  <tr><td colSpan={5} className="subtle" style={{ padding: 20 }}>No entries yet.</td></tr>
                ) : (
                  entries.map((e) => (
                    <tr key={e.id}>
                      <td style={{ whiteSpace: "nowrap" }}>{new Date(e.ts).toLocaleString()}</td>
                      <td>{e.actor_user_id ? e.actor_user_id.slice(0, 8) : <span className="subtle">system</span>}</td>
                      <td><code>{e.action}</code></td>
                      <td>{e.target_type && <span className="subtle">{e.target_type}/{e.target_id?.slice(0, 8) ?? ""}</span>}</td>
                      <td className="mono" style={{ fontSize: 12 }}>{Object.keys(e.metadata || {}).length > 0 ? JSON.stringify(e.metadata) : ""}</td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </>
  );
}
