// SSE passthrough: forward the upstream stream body to the browser unchanged.
const base = process.env.API_BASE ?? "http://localhost:8080/api/v1";

type Params = { params: Promise<{ id: string }> };

export async function GET(_req: Request, { params }: Params) {
  const { id } = await params;
  const upstream = await fetch(`${base}/runs/${id}/logs/stream`, { cache: "no-store" });
  if (!upstream.ok || !upstream.body) {
    return new Response(`upstream error: ${upstream.status}`, { status: 502 });
  }
  return new Response(upstream.body, {
    status: 200,
    headers: {
      "content-type": "text/event-stream",
      "cache-control": "no-cache",
      connection: "keep-alive",
    },
  });
}
