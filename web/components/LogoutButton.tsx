"use client";

import { useRouter } from "next/navigation";
import { IconLogout } from "./icons";

// `variant="nav"` renders as a sidebar nav item; default renders as a button.
export function LogoutButton({ variant = "button" }: { variant?: "button" | "nav" }) {
  const router = useRouter();

  async function logout() {
    await fetch("/api/auth/logout", { method: "POST" });
    router.push("/login");
    router.refresh();
  }

  if (variant === "nav") {
    return (
      <button onClick={logout} className="nav-item" type="button">
        <IconLogout />
        <span>Logout</span>
      </button>
    );
  }

  return (
    <button onClick={logout} className="button secondary sm" type="button">
      Sign out
    </button>
  );
}
