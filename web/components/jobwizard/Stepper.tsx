import { IconCheck } from "@/components/icons";

export type StepDef = { title: string; desc: string };

export function Stepper({ steps, current }: { steps: StepDef[]; current: number }) {
  return (
    <div className="stepper">
      {steps.map((s, i) => {
        const state = i < current ? "done" : i === current ? "current" : "";
        return (
          <div className={`step-item ${state}`} key={s.title}>
            <span className="step-dot">{i < current ? <IconCheck /> : i + 1}</span>
            <span className="step-text">
              <span className="t">{s.title}</span>
              <span className="d">{s.desc}</span>
            </span>
          </div>
        );
      })}
    </div>
  );
}
