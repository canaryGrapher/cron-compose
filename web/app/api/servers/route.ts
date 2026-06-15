// Thin proxy from the Next.js client to the control-plane REST API. Keeps the API
// base URL server-side so the browser never talks to the control plane directly.
import { NextResponse } from "next/server";
import { apiPost } from "@/lib/api";
import type { CreateServerResponse } from "@/lib/types";

export async function POST(req: Request) {
  const body = await req.json();
  try {
    const data = await apiPost<CreateServerResponse>("/servers", body);
    return NextResponse.json(data, { status: 201 });
  } catch (e) {
    return NextResponse.json(
      { error: (e as Error).message },
      { status: 502 },
    );
  }
}
