import Link from "next/link";
import { apiGet } from "@/lib/api";
import type { ListResponse, Server } from "@/lib/types";
import { ServerCard } from "@/components/ServerCard";

export default async function Page() {
  let servers: Server[] = [];
  let error: string | null = null;
  try {
    const data = await apiGet<ListResponse<Server>>("/servers");
    servers = data.items;
  } catch (e) {
    error = (e as Error).message;
  }

  return (
    <>
      <div className="row">
        <div>
          <h1>Servers</h1>
          <p className="subtle">Linux machines running a CronCompose agent.</p>
        </div>
        <Link href="/servers/new" className="button">
          Add server
        </Link>
      </div>

      {error && (
        <div className="panel" style={{ marginTop: 16, color: "var(--danger)" }}>
          Could not reach the control plane: <code>{error}</code>
        </div>
      )}

      {!error && servers.length === 0 && (
        <div className="panel" style={{ marginTop: 16 }}>
          <p className="subtle">No servers yet. Add one to get started.</p>
        </div>
      )}

      <div className="stack" style={{ marginTop: 16 }}>
        {servers.map((s) => (
          <ServerCard key={s.id} server={s} />
        ))}
      </div>
    </>
  );
}
