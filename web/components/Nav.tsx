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
  const isAdmin = me?.role === "admin" || me?.role === "owner";

  return (
    <div className="topbar">
      <Link href="/" className="brand">
        <span className="mark">C</span>
        CronCompose
      </Link>
      <nav>
        {me ? (
          <>
            <Link href="/">Servers</Link>
            {isAdmin && (
              <>
                <Link href="/secrets">Secrets</Link>
                <Link href="/audit">Audit</Link>
              </>
            )}
            <span className="pill" style={{ marginLeft: 4 }}>
              {me.email} · {me.role}
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
