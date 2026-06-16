import type { Me } from "@/lib/types";
import { IconSearch, IconMail, IconBell } from "./icons";

function initials(me: Me): string {
  const src = me.name?.trim() || me.email;
  const parts = src.split(/[\s@._-]+/).filter(Boolean);
  if (parts.length >= 2) return (parts[0][0] + parts[1][0]).toUpperCase();
  return src.slice(0, 2).toUpperCase();
}

export function Topbar({ me }: { me: Me }) {
  const name = me.name?.trim() || me.email.split("@")[0];

  return (
    <header className="topbar">
      <div className="search">
        <IconSearch />
        <input type="search" placeholder="Search servers, jobs, runs…" aria-label="Search" />
        <span className="kbd">⌘ F</span>
      </div>

      <div className="topbar-actions">
        <button className="icon-btn" aria-label="Messages" type="button"><IconMail /></button>
        <button className="icon-btn" aria-label="Notifications" type="button"><IconBell /></button>
        <div className="profile">
          <span className="avatar">{initials(me)}</span>
          <div className="who">
            <div className="name">{name}</div>
            <div className="mail">{me.email}</div>
          </div>
        </div>
      </div>
    </header>
  );
}
