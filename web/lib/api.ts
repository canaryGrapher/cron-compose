// Thin fetch wrapper around the control-plane REST API. Server components forward
// the user's session cookie so authenticated reads work in SSR.
import { cookies } from "next/headers";

const base = process.env.API_BASE ?? "http://localhost:8080/api/v1";

async function call<T>(method: string, path: string, body?: unknown): Promise<T> {
  const cookieJar = await cookies();
  const cookieHeader = cookieJar
    .getAll()
    .map((c) => `${c.name}=${c.value}`)
    .join("; ");

  const headers: Record<string, string> = {};
  if (body !== undefined) headers["content-type"] = "application/json";
  if (cookieHeader) headers["cookie"] = cookieHeader;

  const res = await fetch(`${base}${path}`, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
    cache: "no-store",
  });
  if (!res.ok) {
    const txt = await res.text().catch(() => "");
    throw new Error(`${method} ${path}: ${res.status} ${txt}`);
  }
  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

export const apiGet    = <T>(p: string)               => call<T>("GET", p);
export const apiPost   = <T>(p: string, body: unknown) => call<T>("POST", p, body);
export const apiPatch  = <T>(p: string, body: unknown) => call<T>("PATCH", p, body);
export const apiDelete = <T>(p: string)               => call<T>("DELETE", p);
