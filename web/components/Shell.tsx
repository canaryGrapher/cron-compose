import { apiGet } from "@/lib/api";
import type { Me } from "@/lib/types";
import { Sidebar } from "./Sidebar";
import { Topbar } from "./Topbar";

// Wraps every route. When there's no session (e.g. /login) it renders a bare
// centered shell; otherwise it renders the full sidebar + topbar app frame.
export async function Shell({ children }: { children: React.ReactNode }) {
  let me: Me | null = null;
  try {
    me = await apiGet<Me>("/me");
  } catch {
    me = null;
  }

  if (!me) {
    return <div className="auth-shell">{children}</div>;
  }

  return (
    <div className="app-shell">
      <Sidebar me={me} />
      <div className="app-main">
        <Topbar me={me} />
        <main className="content">{children}</main>
      </div>
    </div>
  );
}
