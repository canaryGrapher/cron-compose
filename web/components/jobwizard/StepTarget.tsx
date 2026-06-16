import type { JobDraft, Patch } from "./types";
import { parseLabels } from "./types";
import { IconServer, IconLayers } from "@/components/icons";

export function StepTarget({
  draft,
  set,
  serverName,
}: {
  draft: JobDraft;
  set: (p: Patch) => void;
  serverName?: string;
}) {
  const labelCount = Object.keys(parseLabels(draft.targetLabels)).length;

  return (
    <div>
      <h2 className="step-h">Where should this job run?</h2>
      <p className="step-lead">Target a single server, or every server matching a set of labels.</p>

      <div className="option-cards">
        <button
          type="button"
          className={`option${draft.targetKind === "server" ? " selected" : ""}`}
          onClick={() => set({ targetKind: "server" })}
        >
          <span className="opt-icon"><IconServer /></span>
          <span>
            <span className="opt-title">This server</span>
            <span className="opt-desc">{serverName ? `Run only on ${serverName}.` : "Run on the selected server only."}</span>
          </span>
        </button>

        <button
          type="button"
          className={`option${draft.targetKind === "labels" ? " selected" : ""}`}
          onClick={() => set({ targetKind: "labels" })}
        >
          <span className="opt-icon"><IconLayers /></span>
          <span>
            <span className="opt-title">Label selector</span>
            <span className="opt-desc">Run on every server whose labels match.</span>
          </span>
        </button>
      </div>

      {draft.targetKind === "labels" && (
        <div className="field" style={{ marginTop: 18 }}>
          <label>Selector (key=value, comma separated)</label>
          <input
            value={draft.targetLabels}
            onChange={(e) => set({ targetLabels: e.target.value })}
            placeholder="env=prod, role=worker"
            autoFocus
          />
          <p className="field-hint">
            {labelCount > 0
              ? `Matches servers with all ${labelCount} label${labelCount > 1 ? "s" : ""}.`
              : "The job runs on every server whose labels contain all of these."}
          </p>
        </div>
      )}
    </div>
  );
}
