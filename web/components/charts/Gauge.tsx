// Semicircular gauge (hand-rolled SVG, no deps). `value` is 0–100.
export type LegendItem = { label: string; value: number; color: string };

const CX = 100;
const CY = 100;
const R = 84;

// Point on the semicircle at progress p (0 = left, 1 = right) over the top.
function pointAt(p: number): [number, number] {
  const angle = Math.PI - p * Math.PI; // radians: PI (left) -> 0 (right)
  return [CX + R * Math.cos(angle), CY - R * Math.sin(angle)];
}

export function Gauge({
  value,
  centerLabel = "",
  legend = [],
}: {
  value: number;
  centerLabel?: string;
  legend?: LegendItem[];
}) {
  const v = Math.max(0, Math.min(100, value));
  const [ex, ey] = pointAt(v / 100);
  const [sx, sy] = pointAt(0);
  const [rx, ry] = pointAt(1);

  return (
    <div className="gauge-wrap">
      <svg viewBox="0 0 200 116" width="100%" style={{ maxWidth: 240 }} role="img" aria-label={`${Math.round(v)} percent`}>
        <path d={`M${sx} ${sy} A${R} ${R} 0 0 1 ${rx} ${ry}`} fill="none" stroke="var(--surface-3)" strokeWidth="18" strokeLinecap="round" />
        {v > 0 && (
          <path d={`M${sx} ${sy} A${R} ${R} 0 0 1 ${ex} ${ey}`} fill="none" stroke="var(--green)" strokeWidth="18" strokeLinecap="round" />
        )}
      </svg>
      <div className="gauge-center">
        <div className="g-value">{Math.round(v)}%</div>
        <div className="g-label">{centerLabel}</div>
      </div>
      {legend.length > 0 && (
        <div className="gauge-legend">
          {legend.map((l) => (
            <span className="lg" key={l.label}>
              <span className="dot" style={{ background: l.color }} />
              {l.label} {l.value}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}
