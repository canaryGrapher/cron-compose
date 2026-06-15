import { forward } from "@/lib/proxy";

type Params = { params: Promise<{ id: string }> };

export async function GET(req: Request, { params }: Params) {
  const { id } = await params;
  return forward(req, `/runs/${id}`);
}
