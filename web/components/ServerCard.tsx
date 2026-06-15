import Link from "next/link";
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
          <div style={{ fontWeight: 600 }}>{server.name}</div>
          <div className="subtle" style={{ fontSize: 12 }}>
            {server.os || "unknown"} / {server.arch || "unknown"}
          </div>
        </div>
        <span
          style={{
            color: statusColor[server.status],
            fontSize: 12,
            textTransform: "uppercase",
            letterSpacing: 0.5,
          }}
        >
          {server.status}
        </span>
      </div>
      {server.description && (
        <p className="subtle" style={{ marginTop: 8 }}>
          {server.description}
        </p>
      )}
    </Link>
  );
}
