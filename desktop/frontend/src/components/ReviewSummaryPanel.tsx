import { useState } from "react";
import { ChevronDown, ChevronUp, GitBranch, Sparkles, Target, Zap } from "lucide-react";
import { ReviewMetrics, normalizeReviewIssue } from "../lib/reviewMetrics";

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
  /** 可折叠模式：默认收起，把空间留给下方报告正文 */
  collapsible?: boolean;
  defaultExpanded?: boolean;
  /** 子 TAB 内嵌：占满剩余高度并可滚动 */
  fill?: boolean;
};

export default function ReviewSummaryPanel({
  metrics,
  collapsible: collapsibleProp,
  defaultExpanded = false,
  fill = false,
}: Props) {
  const collapsible = collapsibleProp ?? !fill;
  const [expanded, setExpanded] = useState(!collapsible || defaultExpanded);
  const { hookScore, coolPoint, debt, issues } = metrics;
  const fixIssues = issues.filter((i) => issueKind(i) === "fix");
  const strengthenIssues = issues.filter((i) => issueKind(i) === "strengthen");
  const otherIssues = issues.filter((i) => issueKind(i) === "other");
  const issueCount = issues.length;

  if (collapsible && !expanded) {
    return (
      <section className="shrink-0 border-b border-studio-border bg-studio-panel/60 px-4 py-2.5">
        <button
          type="button"
          onClick={() => setExpanded(true)}
          className="flex w-full items-center gap-3 text-left transition hover:opacity-90"
        >
          <Sparkles className="h-4 w-4 shrink-0 text-studio-accent" />
          <div className="flex min-w-0 flex-1 flex-wrap items-center gap-x-3 gap-y-1">
            {hookScore != null && (
              <span className={`text-sm font-semibold tabular-nums ${hookScoreColor(hookScore)}`}>
                追读力 {hookScore.toFixed(1)}
              </span>
            )}
            {coolPoint.trim() && (
              <span className="truncate text-xs text-studio-muted">
                爽点：{coolPoint.length > 36 ? `${coolPoint.slice(0, 36)}…` : coolPoint}
              </span>
            )}
            {issueCount > 0 && (
              <span className="rounded-full bg-studio-accent/10 px-2 py-0.5 text-[10px] text-studio-accent">
                {issueCount} 条建议
              </span>
            )}
          </div>
          <span className="inline-flex shrink-0 items-center gap-0.5 text-xs text-studio-accent">
            展开摘要
            <ChevronDown className="h-3.5 w-3.5" />
          </span>
        </button>
      </section>
    );
  }

  return (
    <section
      className={
        fill
          ? "flex min-h-0 flex-1 flex-col overflow-hidden bg-gradient-to-br from-studio-accent/5 via-transparent to-studio-ai/5"
          : "shrink-0 border-b border-studio-border bg-gradient-to-br from-studio-accent/5 via-transparent to-studio-ai/5"
      }
    >
      <div className="flex items-center justify-between gap-2 px-4 py-2.5">
        <div className="flex items-center gap-2">
          <Sparkles className="h-4 w-4 text-studio-accent" />
          <h3 className="text-sm font-medium text-studio-text">审查摘要</h3>
        </div>
        {collapsible && (
          <button
            type="button"
            onClick={() => setExpanded(false)}
            className="inline-flex items-center gap-0.5 rounded-lg px-2 py-1 text-xs text-studio-muted hover:bg-studio-panel hover:text-studio-text"
          >
            收起
            <ChevronUp className="h-3.5 w-3.5" />
          </button>
        )}
      </div>

      <div
        className={`overflow-y-auto px-4 pb-3 ${
          fill ? "min-h-0 flex-1" : collapsible ? "max-h-[min(240px,32vh)]" : "max-h-[min(320px,40vh)]"
        }`}
      >
        <div className="grid gap-2 sm:grid-cols-3">
          <div className="rounded-lg border border-studio-border bg-studio-panel/80 p-2.5">
            <p className="flex items-center gap-1 text-[10px] font-medium uppercase tracking-wide text-studio-muted">
              <Target className="h-3 w-3" />
              追读力
            </p>
            {hookScore != null ? (
              <p className={`mt-0.5 text-xl font-semibold tabular-nums ${hookScoreColor(hookScore)}`}>
                {hookScore.toFixed(1)}
                <span className="ml-1 text-xs font-normal text-studio-muted">/ 10</span>
              </p>
            ) : (
              <p className="mt-0.5 text-xs text-studio-muted">暂无</p>
            )}
          </div>

          <div className="rounded-lg border border-studio-border bg-studio-panel/80 p-2.5 sm:col-span-2">
            <p className="flex items-center gap-1 text-[10px] font-medium uppercase tracking-wide text-studio-muted">
              <Zap className="h-3 w-3" />
              爽点
            </p>
            <p className="mt-0.5 text-xs leading-relaxed text-studio-text">
              {coolPoint.trim() || "未记录"}
            </p>
          </div>

          {debt.trim() && (
            <div className="rounded-lg border border-studio-border bg-studio-panel/80 p-2.5 sm:col-span-3">
              <p className="flex items-center gap-1 text-[10px] font-medium uppercase tracking-wide text-studio-muted">
                <GitBranch className="h-3 w-3" />
                待回收伏笔 / 债务
              </p>
              <p className="mt-0.5 text-xs leading-relaxed text-studio-text">{debt}</p>
            </div>
          )}
        </div>

        {issues.length > 0 && (
          <div className="mt-3 space-y-2">
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
      </div>
    </section>
  );
}

function formatIssueText(item: unknown): string {
  return normalizeReviewIssue(item);
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
      <p className="mb-1 text-[11px] font-medium text-studio-muted">{title}</p>
      <ul className="space-y-1">
        {items.map((item, i) => (
          <li
            key={i}
            className={`rounded-md border px-2.5 py-1.5 text-xs leading-relaxed text-studio-text ${toneClass}`}
          >
            {formatIssueText(item).replace(/^【(?:修正|加强)】/, "").trim() || formatIssueText(item)}
          </li>
        ))}
      </ul>
    </div>
  );
}
