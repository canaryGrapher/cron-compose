import Link from "next/link";
import { apiGet } from "@/lib/api";
import { LogoutButton } from "./LogoutButton";

type Me = { id: string; email: string; name: string; role: string };

export async function Nav() {
  let me: Me | null = null;
  try {
    me = await apiGet<Me>("/me");
  } catch {
    me = null;
  }
  return (
    <div className="row" style={{ marginBottom: 24 }}>
      <Link href="/" style={{ fontWeight: 600, color: "var(--text)" }}>
        CronCompose
      </Link>
      <nav style={{ display: "flex", gap: 16, alignItems: "center" }}>
        {me ? (
          <>
            <Link href="/">Servers</Link>
            {(me.role === "admin" || me.role === "owner") && (
              <>
                <Link href="/secrets">Secrets</Link>
                <Link href="/audit">Audit</Link>
              </>
            )}
            <span className="subtle" style={{ fontSize: 12 }}>
              {me.email} ({me.role})
            </span>
            <LogoutButton />
          </>
        ) : (
          <Link href="/login">Sign in</Link>
        )}
      </nav>
    </div>
  );
}
