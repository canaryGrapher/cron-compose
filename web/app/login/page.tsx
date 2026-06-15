"use client";

import { Suspense, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";

type AuthConfig = {
  password_login: boolean;
  oidc_enabled: boolean;
  oidc_start_url: string;
};

function LoginForm() {
  const router = useRouter();
  const params = useSearchParams();
  const next = params.get("next") || "/";

  const [authCfg, setAuthCfg] = useState<AuthConfig | null>(null);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch("/api/auth/config")
      .then((r) => r.json() as Promise<AuthConfig>)
      .then(setAuthCfg)
      .catch(() => setAuthCfg({ password_login: true, oidc_enabled: false, oidc_start_url: "" }));
  }, []);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const res = await fetch("/api/auth/login", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ email, password }),
      });
      if (!res.ok) {
        if (res.status === 401) throw new Error("Wrong email or password");
        throw new Error(`HTTP ${res.status}`);
      }
      router.push(next);
      router.refresh();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }

  function startSSO() {
    if (!authCfg?.oidc_start_url) return;
    // Hand the browser to the control plane directly so the OIDC redirect chain stays
    // server-to-server clean.
    const u = new URL(authCfg.oidc_start_url, window.location.origin);
    u.searchParams.set("next", next);
    window.location.href = u.toString();
  }

  return (
    <>
      <h1>Sign in</h1>
      <p className="subtle">CronCompose control plane.</p>

      {authCfg?.oidc_enabled && (
        <div className="stack" style={{ marginTop: 16, maxWidth: 360 }}>
          <button className="button" onClick={startSSO} type="button">
            Sign in with SSO
          </button>
          <div className="subtle" style={{ textAlign: "center", fontSize: 12, margin: "8px 0" }}>
            or with email + password
          </div>
        </div>
      )}

      <form onSubmit={submit} className="stack" style={{ marginTop: 16, maxWidth: 360 }}>
        <div>
          <label htmlFor="email">Email</label>
          <input
            id="email"
            type="email"
            autoComplete="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
          />
        </div>
        <div>
          <label htmlFor="password">Password</label>
          <input
            id="password"
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
        </div>
        {error && <p style={{ color: "var(--danger)" }}>{error}</p>}
        <div>
          <button type="submit" className="button" disabled={busy || !email || !password}>
            {busy ? "Signing in..." : "Sign in"}
          </button>
        </div>
      </form>
    </>
  );
}

// useSearchParams() must sit inside a Suspense boundary or Next.js 16 fails to
// statically prerender /login (CSR bailout). The boundary keeps the build happy.
export default function LoginPage() {
  return (
    <Suspense fallback={null}>
      <LoginForm />
    </Suspense>
  );
}
