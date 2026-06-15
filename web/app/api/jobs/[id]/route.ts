import { forward } from "@/lib/proxy";

type Params = { params: Promise<{ id: string }> };

export async function GET(req: Request, { params }: Params) {
  const { id } = await params;
  return forward(req, `/jobs/${id}`);
}

export async function PATCH(req: Request, { params }: Params) {
  const { id } = await params;
  return forward(req, `/jobs/${id}`);
}

export async function DELETE(req: Request, { params }: Params) {
  const { id } = await params;
  return forward(req, `/jobs/${id}`);
}
