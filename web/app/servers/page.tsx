import Link from "next/link";
import { apiGet } from "@/lib/api";
import type { ListResponse, Server } from "@/lib/types";
import { ServerCard } from "@/components/ServerCard";
import { IconPlus } from "@/components/icons";

export default async function ServersPage() {
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
      <div className="page-head">
        <div>
          <h1>Servers</h1>
          <p className="subtle">Linux machines running a CronCompose agent.</p>
        </div>
        <div className="page-head-actions">
          <Link href="/servers/new" className="button"><IconPlus /> Add server</Link>
        </div>
      </div>

      {error && (
        <div className="form-error">Could not reach the control plane: <code>{error}</code></div>
      )}

      {!error && servers.length === 0 && (
        <div className="panel"><div className="empty">No servers yet. Add one to get started.</div></div>
      )}

      <div className="cards">
        {servers.map((s) => <ServerCard key={s.id} server={s} />)}
      </div>
    </>
  );
}
