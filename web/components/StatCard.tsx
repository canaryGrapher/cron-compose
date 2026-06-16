import Link from "next/link";
import { IconArrowUpRight } from "./icons";

export type StatChip = { text: string; tone?: "up" | "down" | "neutral" };

export function StatCard({
  label,
  value,
  href,
  foot,
  chip,
  green = false,
}: {
  label: string;
  value: number | string;
  href: string;
  foot: string;
  chip?: StatChip;
  green?: boolean;
}) {
  return (
    <Link href={href} className={`stat${green ? " green" : ""}`}>
      <div className="stat-top">
        <span className="stat-label">{label}</span>
        <span className="stat-arrow"><IconArrowUpRight /></span>
      </div>
      <div className="stat-value">{value}</div>
      <div className="stat-foot">
        {chip && <span className={`trend ${chip.tone ?? "neutral"}`}>{chip.text}</span>}
        <span>{foot}</span>
      </div>
    </Link>
  );
}
