// Generic helper for Next.js route handlers that forward a single request to the
// control-plane API, passing the session cookie through both ways.

const base = process.env.API_BASE ?? "http://localhost:8080/api/v1";

export async function forward(req: Request, path: string, method?: string) {
  const m = method ?? req.method;
  const body = m === "GET" || m === "DELETE" ? undefined : await req.text();

  const headers: Record<string, string> = {};
  if (body) headers["content-type"] = "application/json";
  const incomingCookie = req.headers.get("cookie");
  if (incomingCookie) headers["cookie"] = incomingCookie;

  const res = await fetch(`${base}${path}`, {
    method: m,
    headers,
    body,
    cache: "no-store",
  });
  const text = await res.text();

  const outHeaders: Record<string, string> = {
    "content-type": res.headers.get("content-type") ?? "application/json",
  };
  const setCookie = res.headers.get("set-cookie");
  if (setCookie) outHeaders["set-cookie"] = setCookie;

  return new Response(text, { status: res.status, headers: outHeaders });
}
