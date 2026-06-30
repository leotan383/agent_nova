import { Check } from "lucide-react";

const STEPS = [
  { id: "gate", label: "检查" },
  { id: "context", label: "上下文" },
  { id: "taskbook", label: "任务书" },
  { id: "draft", label: "起草" },
  { id: "review", label: "润色" },
  { id: "summary", label: "摘要" },
  { id: "extract", label: "记忆" },
  { id: "done", label: "完成" },
] as const;

type Props = {
  currentStep: string;
  running: boolean;
};

function stepIndex(id: string): number {
  if (!id) return -1;
  const i = STEPS.findIndex((s) => s.id === id);
  return i;
}

export default function WriteStepper({ currentStep, running }: Props) {
  const activeIdx = stepIndex(currentStep);
  const displayIdx = activeIdx >= 0 ? activeIdx : running ? 0 : -1;

  if (!running && displayIdx < 0) {
    return (
      <div className="shrink-0 rounded-lg border border-dashed border-studio-border/80 bg-studio-bg/40 px-4 py-2.5 text-xs text-studio-muted">
        开始写章后，流水线进度将显示在这里
      </div>
    );
  }

  return (
    <div className="shrink-0 overflow-x-auto rounded-lg border border-studio-border bg-studio-panel px-3 py-3">
      <ol className="flex min-w-max items-center gap-1">
        {STEPS.map((s, i) => {
          const finished = currentStep === "done";
          const done = finished || displayIdx > i;
          const active = !finished && (s.id === currentStep || (running && displayIdx === i));
          const pending = !finished && displayIdx >= 0 && i > displayIdx;

          return (
            <li key={s.id} className="flex items-center">
              {i > 0 && (
                <div
                  className={`mx-1 h-px w-4 sm:w-6 ${
                    done ? "bg-studio-accent/50" : "bg-studio-border"
                  }`}
                />
              )}
              <div
                className={`flex items-center gap-1.5 rounded-full px-2 py-1 text-[11px] font-medium transition sm:px-2.5 ${
                  active
                    ? "bg-studio-accent/15 text-studio-accent ring-1 ring-studio-accent/30"
                    : done
                      ? "text-[rgb(var(--studio-diff-add-stat))]"
                      : pending
                        ? "text-studio-muted/50"
                        : "text-studio-muted"
                }`}
              >
                <span
                  className={`flex h-4 w-4 shrink-0 items-center justify-center rounded-full text-[9px] ${
                    active
                      ? "bg-studio-accent text-studio-on-accent"
                      : done
                        ? "bg-[rgb(var(--studio-diff-add-bg))]"
                        : "border border-studio-border bg-studio-bg"
                  }`}
                >
                  {done && !active ? <Check className="h-2.5 w-2.5" /> : i + 1}
                </span>
                <span className="hidden sm:inline">{s.label}</span>
              </div>
            </li>
          );
        })}
      </ol>
    </div>
  );
}
