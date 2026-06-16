"use client";

import { useRouter } from "next/navigation";

export function LogoutButton() {
  const router = useRouter();
  async function logout() {
    await fetch("/api/auth/logout", { method: "POST" });
    router.push("/login");
    router.refresh();
  }
  return (
    <button
      onClick={logout}
      className="button secondary"
      style={{ fontSize: 12, padding: "7px 14px" }}
    >
      Sign out
    </button>
  );
}
