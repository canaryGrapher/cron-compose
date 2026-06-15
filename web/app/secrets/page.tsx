"use client";

import { useEffect, useState } from "react";
import type { ListResponse, Secret } from "@/lib/types";

export default function SecretsPage() {
  const [items, setItems] = useState<Secret[]>([]);
  const [name, setName] = useState("");
  const [value, setValue] = useState("");
  const [scope, setScope] = useState<"global" | "server" | "job">("global");
  const [scopeID, setScopeID] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function load() {
    try {
      const res = await fetch("/api/secrets").then((r) => r.json() as Promise<ListResponse<Secret>>);
      setItems(res.items);
    } catch (e) {
      setError((e as Error).message);
    }
  }
  useEffect(() => {
    load();
  }, []);

  async function create(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const res = await fetch("/api/secrets", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({
          name,
          value,
          scope,
          scope_id: scopeID || undefined,
        }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setName("");
      setValue("");
      setScopeID("");
      await load();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }

  async function remove(id: string) {
    if (!confirm("Delete this secret? Jobs that reference it will lose the value on next sync.")) return;
    await fetch(`/api/secrets/${id}`, { method: "DELETE" });
    await load();
  }

  return (
    <>
      <h1>Secrets</h1>
      <p className="subtle">
        Values are write-only. Reference a secret by name in a job's <code>secret_refs</code>;
        the agent receives it as an environment variable at run time.
      </p>

      <h2>Add a secret</h2>
      <form onSubmit={create} className="stack" style={{ maxWidth: 540 }}>
        <div className="row" style={{ gap: 12 }}>
          <div style={{ flex: 1 }}>
            <label>Name (env var)</label>
            <input value={name} onChange={(e) => setName(e.target.value)} placeholder="API_KEY" required />
          </div>
          <div style={{ flex: 1 }}>
            <label>Scope</label>
            <select
              value={scope}
              onChange={(e) => setScope(e.target.value as typeof scope)}
              style={{
                width: "100%",
                background: "var(--bg)",
                color: "var(--text)",
                border: "1px solid var(--border)",
                borderRadius: 6,
                padding: "8px 10px",
              }}
            >
              <option value="global">global</option>
              <option value="server">server</option>
              <option value="job">job</option>
            </select>
          </div>
        </div>
        {scope !== "global" && (
          <div>
            <label>Scope ID ({scope})</label>
            <input value={scopeID} onChange={(e) => setScopeID(e.target.value)} placeholder={`${scope}_id`} required />
          </div>
        )}
        <div>
          <label>Value</label>
          <input type="password" value={value} onChange={(e) => setValue(e.target.value)} required />
        </div>
        {error && <p style={{ color: "var(--danger)" }}>{error}</p>}
        <div>
          <button type="submit" className="button" disabled={busy || !name || !value}>
            {busy ? "Adding..." : "Add secret"}
          </button>
        </div>
      </form>

      <h2>Existing secrets</h2>
      {items.length === 0 ? (
        <div className="panel">
          <p className="subtle">No secrets yet.</p>
        </div>
      ) : (
        <div className="stack">
          {items.map((s) => (
            <div key={s.id} className="panel">
              <div className="row">
                <div>
                  <div style={{ fontWeight: 600 }}>{s.name}</div>
                  <div className="subtle" style={{ fontSize: 12 }}>
                    scope: {s.scope}
                    {s.scope_id ? ` (${s.scope_id.slice(0, 8)})` : ""} · created {new Date(s.created_at).toLocaleString()}
                  </div>
                </div>
                <button className="button secondary" onClick={() => remove(s.id)}>
                  Delete
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </>
  );
}
