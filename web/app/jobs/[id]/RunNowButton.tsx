"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import type { RunNowResult } from "@/lib/types";

export function RunNowButton({ jobId }: { jobId: string }) {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [last, setLast] = useState<RunNowResult["runs"] | null>(null);

  async function trigger() {
    setBusy(true);
    setError(null);
    try {
      const res = await fetch(`/api/jobs/${jobId}/run`, { method: "POST" });
      if (!res.ok) {
        if (res.status === 503) throw new Error("Agent is offline");
        if (res.status === 400) throw new Error("No servers match this job");
        throw new Error(`HTTP ${res.status}`);
      }
      const data = (await res.json()) as RunNowResult;
      setLast(data.runs);
      if (data.runs.length === 1) {
        router.push(`/runs/${data.runs[0].run_id}`);
      }
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div style={{ textAlign: "right" }}>
      <button className="button" onClick={trigger} disabled={busy}>
        {busy ? "Triggering..." : "Run now"}
      </button>
      {error && (
        <p style={{ color: "var(--danger)", fontSize: 12, marginTop: 6 }}>{error}</p>
      )}
      {last && last.length > 1 && (
        <div style={{ marginTop: 8, textAlign: "left", fontSize: 12 }} className="subtle">
          {last.length} runs queued:
          <ul style={{ margin: "4px 0 0 16px" }}>
            {last.map((r) => (
              <li key={r.run_id}>
                <Link href={`/runs/${r.run_id}`}>{r.run_id.slice(0, 8)}</Link>{" "}
                ({r.server_id.slice(0, 8)}) — {r.status}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
