import { forward } from "@/lib/proxy";

type Params = { params: Promise<{ id: string }> };

export async function POST(req: Request, { params }: Params) {
  const { id } = await params;
  return forward(req, `/jobs/${id}/run`);
}
