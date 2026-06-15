"use client";

import { useState, use } from "react";
import { useRouter } from "next/navigation";
import type { Job } from "@/lib/types";

type Props = { params: Promise<{ id: string }> };

export default function NewJobPage({ params }: Props) {
  const { id: serverID } = use(params);
  const router = useRouter();

  const [targetKind, setTargetKind] = useState<"server" | "labels">("server");
  const [targetLabels, setTargetLabels] = useState(""); // "k=v, k2=v2"
  const [name, setName] = useState("");
  const [scheduleCron, setScheduleCron] = useState("0 */6 * * *");
  const [timezone, setTimezone] = useState("UTC");
  const [interpreter, setInterpreter] = useState("bash");
  const [scriptBody, setScriptBody] = useState(
    "#!/usr/bin/env bash\nset -euo pipefail\n\necho \"hello from cron\"\n",
  );
  const [timeoutSeconds, setTimeoutSeconds] = useState(3600);
  const [cpuPct, setCPUPct] = useState(0);
  const [memMB, setMemMB] = useState(0);
  const [secretRefs, setSecretRefs] = useState(""); // comma separated names
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function parseLabels(input: string): Record<string, string> {
    const out: Record<string, string> = {};
    for (const part of input.split(",")) {
      const trimmed = part.trim();
      if (!trimmed) continue;
      const eq = trimmed.indexOf("=");
      if (eq < 0) continue;
      out[trimmed.slice(0, eq).trim()] = trimmed.slice(eq + 1).trim();
    }
    return out;
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const body: Record<string, unknown> = {
        target_kind: targetKind,
        name,
        schedule_cron: scheduleCron,
        timezone,
        interpreter,
        script_body: scriptBody,
        timeout_seconds: timeoutSeconds,
        cpu_quota_percent: cpuPct,
        memory_max_mb: memMB,
      };
      if (targetKind === "server") {
        body.server_id = serverID;
      } else {
        body.target_labels = parseLabels(targetLabels);
      }
      const refs = secretRefs.split(",").map((s) => s.trim()).filter(Boolean);
      if (refs.length > 0) body.secret_refs = refs;

      const res = await fetch("/api/jobs", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const txt = await res.text().catch(() => "");
        throw new Error(`HTTP ${res.status}: ${txt}`);
      }
      const job = (await res.json()) as Job;
      router.push(`/jobs/${job.id}`);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <h1>New job</h1>
      <p className="subtle">Pick a target, write a script, set a schedule.</p>
      <form onSubmit={submit} className="stack" style={{ marginTop: 16 }}>
        <div className="panel">
          <label>Target</label>
          <div style={{ display: "flex", gap: 16, marginTop: 4 }}>
            <label style={{ display: "flex", alignItems: "center", gap: 6, color: "var(--text)" }}>
              <input
                type="radio"
                name="kind"
                checked={targetKind === "server"}
                onChange={() => setTargetKind("server")}
              />
              this server
            </label>
            <label style={{ display: "flex", alignItems: "center", gap: 6, color: "var(--text)" }}>
              <input
                type="radio"
                name="kind"
                checked={targetKind === "labels"}
                onChange={() => setTargetKind("labels")}
              />
              label selector (multi-server)
            </label>
          </div>
          {targetKind === "labels" && (
            <div style={{ marginTop: 8 }}>
              <label>Selector (key=value, comma separated)</label>
              <input
                value={targetLabels}
                onChange={(e) => setTargetLabels(e.target.value)}
                placeholder="env=prod, role=worker"
              />
              <p className="subtle" style={{ fontSize: 12, marginTop: 4 }}>
                The job runs on every server whose labels contain all of these.
              </p>
            </div>
          )}
        </div>

        <div className="row" style={{ gap: 12 }}>
          <div style={{ flex: 2 }}>
            <label>Name</label>
            <input value={name} onChange={(e) => setName(e.target.value)} placeholder="backup-photos" required />
          </div>
          <div style={{ flex: 1 }}>
            <label>Interpreter</label>
            <input value={interpreter} onChange={(e) => setInterpreter(e.target.value)} />
          </div>
        </div>
        <div className="row" style={{ gap: 12 }}>
          <div style={{ flex: 2 }}>
            <label>Cron schedule</label>
            <input value={scheduleCron} onChange={(e) => setScheduleCron(e.target.value)} required />
          </div>
          <div style={{ flex: 1 }}>
            <label>Timezone</label>
            <input value={timezone} onChange={(e) => setTimezone(e.target.value)} />
          </div>
          <div style={{ flex: 1 }}>
            <label>Timeout (s)</label>
            <input
              type="number"
              value={timeoutSeconds}
              onChange={(e) => setTimeoutSeconds(Number(e.target.value))}
            />
          </div>
        </div>
        <div className="row" style={{ gap: 12 }}>
          <div style={{ flex: 1 }}>
            <label>CPU quota (% of one core, 0 = unlimited)</label>
            <input type="number" min={0} value={cpuPct} onChange={(e) => setCPUPct(Number(e.target.value))} />
          </div>
          <div style={{ flex: 1 }}>
            <label>Memory limit (MB, 0 = unlimited)</label>
            <input type="number" min={0} value={memMB} onChange={(e) => setMemMB(Number(e.target.value))} />
          </div>
        </div>
        <div>
          <label>Secret refs (env var names, comma separated)</label>
          <input
            value={secretRefs}
            onChange={(e) => setSecretRefs(e.target.value)}
            placeholder="API_KEY, DB_PASSWORD"
          />
          <p className="subtle" style={{ fontSize: 12, marginTop: 4 }}>
            The agent receives matching secrets as environment variables at run time.
          </p>
        </div>
        <div>
          <label>Script</label>
          <textarea
            value={scriptBody}
            onChange={(e) => setScriptBody(e.target.value)}
            rows={14}
            style={{ fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace", fontSize: 13 }}
          />
        </div>
        {error && <p style={{ color: "var(--danger)" }}>{error}</p>}
        <div>
          <button type="submit" className="button" disabled={busy || !name || !scheduleCron}>
            {busy ? "Creating..." : "Create job"}
          </button>
        </div>
      </form>
    </>
  );
}
