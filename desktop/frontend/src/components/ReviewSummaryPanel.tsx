import { GitBranch, Sparkles, Target, Zap } from "lucide-react";
import { ReviewMetrics } from "../lib/reviewMetrics";

function issueKind(text: string): "fix" | "strengthen" | "other" {
  if (text.includes("【修正】") || text.includes("[修正]")) return "fix";
  if (text.includes("【加强】") || text.includes("[加强]")) return "strengthen";
  return "other";
}

function hookScoreColor(score: number): string {
  if (score >= 8) return "text-[rgb(var(--studio-diff-add-stat))]";
  if (score >= 5) return "text-studio-accent";
  return "text-[rgb(var(--studio-warning-fg))]";
}

type Props = {
  metrics: ReviewMetrics;
  compact?: boolean;
};

export default function ReviewSummaryPanel({ metrics, compact }: Props) {
  const { hookScore, coolPoint, debt, issues } = metrics;
  const fixIssues = issues.filter((i) => issueKind(i) === "fix");
  const strengthenIssues = issues.filter((i) => issueKind(i) === "strengthen");
  const otherIssues = issues.filter((i) => issueKind(i) === "other");

  return (
    <section
      className={`shrink-0 border-b border-studio-border bg-gradient-to-br from-studio-accent/5 via-transparent to-studio-ai/5 ${
        compact ? "px-3 py-3" : "px-4 py-4"
      }`}
    >
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <Sparkles className="h-4 w-4 text-studio-accent" />
        <h3 className="text-sm font-medium text-studio-text">审查摘要</h3>
      </div>

      <div className="grid gap-3 sm:grid-cols-3">
        <div className="rounded-xl border border-studio-border bg-studio-panel/80 p-3">
          <p className="flex items-center gap-1.5 text-[10px] font-medium uppercase tracking-wide text-studio-muted">
            <Target className="h-3 w-3" />
            追读力
          </p>
          {hookScore != null ? (
            <p className={`mt-1 text-2xl font-semibold tabular-nums ${hookScoreColor(hookScore)}`}>
              {hookScore.toFixed(1)}
              <span className="ml-1 text-sm font-normal text-studio-muted">/ 10</span>
            </p>
          ) : (
            <p className="mt-1 text-sm text-studio-muted">暂无评分</p>
          )}
        </div>

        <div className="rounded-xl border border-studio-border bg-studio-panel/80 p-3 sm:col-span-2">
          <p className="flex items-center gap-1.5 text-[10px] font-medium uppercase tracking-wide text-studio-muted">
            <Zap className="h-3 w-3" />
            爽点
          </p>
          <p className="mt-1 text-sm leading-relaxed text-studio-text">
            {coolPoint.trim() || "未记录"}
          </p>
        </div>

        {debt.trim() && (
          <div className="rounded-xl border border-studio-border bg-studio-panel/80 p-3 sm:col-span-3">
            <p className="flex items-center gap-1.5 text-[10px] font-medium uppercase tracking-wide text-studio-muted">
              <GitBranch className="h-3 w-3" />
              待回收伏笔 / 债务
            </p>
            <p className="mt-1 text-sm leading-relaxed text-studio-text">{debt}</p>
          </div>
        )}
      </div>

      {issues.length > 0 && (
        <div className="mt-4 space-y-3">
          {fixIssues.length > 0 && (
            <IssueGroup title="修正建议" tone="fix" items={fixIssues} />
          )}
          {strengthenIssues.length > 0 && (
            <IssueGroup title="加强建议" tone="strengthen" items={strengthenIssues} />
          )}
          {otherIssues.length > 0 && (
            <IssueGroup title="其他问题" tone="other" items={otherIssues} />
          )}
        </div>
      )}
    </section>
  );
}

function IssueGroup({
  title,
  tone,
  items,
}: {
  title: string;
  tone: "fix" | "strengthen" | "other";
  items: string[];
}) {
  const toneClass =
    tone === "fix"
      ? "border-[rgb(var(--studio-danger-border))] bg-[rgb(var(--studio-danger-bg))]"
      : tone === "strengthen"
        ? "border-[rgb(var(--studio-warning-border))] bg-[rgb(var(--studio-warning-bg))]"
        : "border-studio-border bg-studio-panel/60";

  return (
    <div>
      <p className="mb-1.5 text-xs font-medium text-studio-muted">{title}</p>
      <ul className="space-y-1.5">
        {items.map((item, i) => (
          <li
            key={i}
            className={`rounded-lg border px-3 py-2 text-sm leading-relaxed text-studio-text ${toneClass}`}
          >
            {item.replace(/^【(?:修正|加强)】/, "").trim() || item}
          </li>
        ))}
      </ul>
    </div>
  );
}
