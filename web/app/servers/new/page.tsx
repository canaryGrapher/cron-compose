"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import type { CreateServerResponse } from "@/lib/types";

export default function NewServerPage() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<CreateServerResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const res = await fetch("/api/servers", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ name, description }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = (await res.json()) as CreateServerResponse;
      setResult(data);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }

  if (result) {
    return (
      <>
        <h1>Server created</h1>
        <p className="subtle">
          Run the install command on <strong>{result.server.name}</strong>. The token is
          shown once.
        </p>
        <div className="panel" style={{ marginTop: 12 }}>
          <label>Install command</label>
          <code style={{ display: "block", whiteSpace: "pre-wrap" }}>
            {result.install_command}
          </code>
        </div>
        <div style={{ marginTop: 16 }}>
          <button className="button" onClick={() => router.push("/")}>
            Done
          </button>
        </div>
      </>
    );
  }

  return (
    <>
      <h1>Add server</h1>
      <p className="subtle">Give it a name. You will get an install command next.</p>
      <form onSubmit={submit} className="stack" style={{ marginTop: 16, maxWidth: 480 }}>
        <div>
          <label htmlFor="name">Name</label>
          <input
            id="name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="kitchen-pi"
            required
          />
        </div>
        <div>
          <label htmlFor="description">Description (optional)</label>
          <input
            id="description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Raspberry Pi behind the kitchen TV"
          />
        </div>
        {error && <p style={{ color: "var(--danger)" }}>{error}</p>}
        <div>
          <button type="submit" className="button" disabled={busy || !name}>
            {busy ? "Creating..." : "Create server"}
          </button>
        </div>
      </form>
    </>
  );
}
