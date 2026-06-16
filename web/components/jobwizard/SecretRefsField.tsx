"use client";

import { useEffect, useState } from "react";
import type { ListResponse, Secret } from "@/lib/types";

// Lets the user pick secret names to inject as env vars. Loads existing secrets
// as toggle chips and allows typing custom names too.
export function SecretRefsField({ value, onChange }: { value: string[]; onChange: (next: string[]) => void }) {
  const [available, setAvailable] = useState<string[]>([]);
  const [custom, setCustom] = useState("");

  useEffect(() => {
    fetch("/api/secrets")
      .then((r) => (r.ok ? (r.json() as Promise<ListResponse<Secret>>) : Promise.reject()))
      .then((d) => setAvailable(Array.from(new Set((d.items ?? []).map((s) => s.name)))))
      .catch(() => setAvailable([]));
  }, []);

  const all = Array.from(new Set([...available, ...value]));

  function toggle(name: string) {
    onChange(value.includes(name) ? value.filter((x) => x !== name) : [...value, name]);
  }
  function addCustom() {
    const n = custom.trim();
    if (n && !value.includes(n)) onChange([...value, n]);
    setCustom("");
  }

  return (
    <div>
      {all.length > 0 && (
        <div className="chips" style={{ marginBottom: 10 }}>
          {all.map((n) => (
            <button
              type="button"
              key={n}
              className={`chip${value.includes(n) ? " selected" : ""}`}
              onClick={() => toggle(n)}
            >
              {n}
            </button>
          ))}
        </div>
      )}
      <div className="cluster" style={{ flexWrap: "nowrap" }}>
        <input
          value={custom}
          onChange={(e) => setCustom(e.target.value)}
          onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); addCustom(); } }}
          placeholder="ADD_CUSTOM_VAR"
        />
        <button type="button" className="button secondary sm" onClick={addCustom} disabled={!custom.trim()}>Add</button>
      </div>
      <p className="field-hint">The agent receives each selected secret as an environment variable at run time.</p>
    </div>
  );
}
