// Lightweight CSS bar chart (no SVG, no deps). Highlights one bar (e.g. today)
// in solid green and flags the peak bar with its value.
export type Bar = { label: string; value: number };

export function BarChart({ data, highlightIndex }: { data: Bar[]; highlightIndex?: number }) {
  const max = Math.max(1, ...data.map((d) => d.value));

  return (
    <div className="bars">
      {data.map((d, i) => {
        const pct = Math.round((d.value / max) * 100);
        const isPeak = d.value === max && max > 0;
        const isHot = i === highlightIndex;
        const cls = isHot ? "bar hot" : isPeak ? "bar lit" : "bar";
        return (
          <div className="bar-col" key={i}>
            <div className="bar-track">
              <div className={cls} style={{ height: `${Math.max(pct, 6)}%` }}>
                {isPeak && d.value > 0 && <span className="bar-flag">{d.value}</span>}
              </div>
            </div>
            <span className="bar-label">{d.label}</span>
          </div>
        );
      })}
    </div>
  );
}
