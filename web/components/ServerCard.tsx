import Link from "next/link";
import type { CSSProperties } from "react";
import type { Server } from "@/lib/types";

const statusColor: Record<Server["status"], string> = {
  pending: "var(--muted)",
  online: "var(--ok)",
  offline: "var(--danger)",
};

export function ServerCard({ server }: { server: Server }) {
  return (
    <Link
      href={`/servers/${server.id}`}
      className="panel"
      style={{ display: "block", color: "var(--text)" }}
    >
      <div className="row">
        <div>
          <div style={{ fontWeight: 700, fontSize: 15 }}>{server.name}</div>
          <div className="subtle" style={{ fontSize: 12, marginTop: 2 }}>
            {server.os || "unknown"} / {server.arch || "unknown"}
          </div>
        </div>
        <span className="status" style={{ "--status-color": statusColor[server.status] } as CSSProperties}>
          {server.status}
        </span>
      </div>
      {server.description && (
        <p className="subtle" style={{ marginTop: 10 }}>
          {server.description}
        </p>
      )}
    </Link>
  );
}
