"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import type { ReactNode } from "react";

// A sidebar nav link that highlights when its route (or a child route) is active.
// usePathname() returns the path with the basePath already stripped by Next, so
// we compare against plain hrefs like "/" and "/servers".
export function NavLink({
  href,
  icon,
  children,
  badge,
}: {
  href: string;
  icon: ReactNode;
  children: ReactNode;
  badge?: ReactNode;
}) {
  const pathname = usePathname();
  const active = href === "/" ? pathname === "/" : pathname === href || pathname.startsWith(href + "/");

  return (
    <Link href={href} className={`nav-item${active ? " active" : ""}`}>
      {icon}
      <span>{children}</span>
      {badge !== undefined && <span className="nav-badge">{badge}</span>}
    </Link>
  );
}
