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
    const data = await apiGet<ListResponse<AuditEntry>>("/audit?limit=100");
    entries = data.items;
  } catch (e) {
    error = (e as Error).message;
  }

  return (
    <>
      <h1>Audit log</h1>
      <p className="subtle">Most recent 100 entries. Admin only.</p>

      {error && (
        <div className="panel" style={{ color: "var(--danger)", marginTop: 16 }}>
          Could not load: <code>{error}</code>
        </div>
      )}

      {!error && (
        <div className="panel" style={{ marginTop: 16, padding: 0 }}>
          <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
            <thead>
              <tr style={{ textAlign: "left", color: "var(--muted)" }}>
                <th style={{ padding: "8px 12px" }}>When</th>
                <th style={{ padding: "8px 12px" }}>Actor</th>
                <th style={{ padding: "8px 12px" }}>Action</th>
                <th style={{ padding: "8px 12px" }}>Target</th>
                <th style={{ padding: "8px 12px" }}>Metadata</th>
              </tr>
            </thead>
            <tbody>
              {entries.length === 0 ? (
                <tr>
                  <td colSpan={5} style={{ padding: 16 }} className="subtle">
                    No entries yet.
                  </td>
                </tr>
              ) : (
                entries.map((e) => (
                  <tr key={e.id} style={{ borderTop: "1px solid var(--border)" }}>
                    <td style={{ padding: "8px 12px", whiteSpace: "nowrap" }}>
                      {new Date(e.ts).toLocaleString()}
                    </td>
                    <td style={{ padding: "8px 12px" }}>
                      {e.actor_user_id ? e.actor_user_id.slice(0, 8) : <span className="subtle">system</span>}
                    </td>
                    <td style={{ padding: "8px 12px" }}>
                      <code>{e.action}</code>
                    </td>
                    <td style={{ padding: "8px 12px" }}>
                      {e.target_type && (
                        <span>
                          {e.target_type}/{e.target_id?.slice(0, 8) ?? ""}
                        </span>
                      )}
                    </td>
                    <td style={{ padding: "8px 12px", fontFamily: "ui-monospace, monospace" }}>
                      {Object.keys(e.metadata || {}).length > 0 ? JSON.stringify(e.metadata) : ""}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}
    </>
  );
}
