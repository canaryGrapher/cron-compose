import { forward } from "@/lib/proxy";

export async function GET(req: Request)  { return forward(req, "/secrets"); }
export async function POST(req: Request) { return forward(req, "/secrets"); }
