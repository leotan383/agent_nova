import { useState } from "react";
import { AlertTriangle, Bot, ChevronDown, ChevronUp, ShieldCheck } from "lucide-react";
import { AIDetectMetrics, aiScoreColor, riskLevelColor, riskLevelLabel } from "../lib/aiDetectMetrics";

type Props = {
  metrics: AIDetectMetrics;
  collapsible?: boolean;
  defaultExpanded?: boolean;
  fill?: boolean;
};

export default function AIDetectSummaryPanel({
  metrics,
  collapsible: collapsibleProp,
  defaultExpanded = false,
  fill = false,
}: Props) {
  const collapsible = collapsibleProp ?? !fill;
  const [expanded, setExpanded] = useState(!collapsible || defaultExpanded);
  const { aiScore, humanScore, riskLevel, publishable, signals, hotspots } = metrics;
  const risk = riskLevelLabel(riskLevel);

  if (collapsible && !expanded) {
    return (
      <section className="shrink-0 border-b border-studio-border bg-studio-panel/60 px-4 py-2.5">
        <button
          type="button"
          onClick={() => setExpanded(true)}
          className="flex w-full items-center gap-3 text-left transition hover:opacity-90"
        >
          <Bot className="h-4 w-4 shrink-0 text-studio-ai" />
          <div className="flex min-w-0 flex-1 flex-wrap items-center gap-x-3 gap-y-1">
            {aiScore != null && (
              <span className={`text-sm font-semibold tabular-nums ${aiScoreColor(aiScore)}`}>
                AI 味 {aiScore.toFixed(1)}
              </span>
            )}
            {riskLevel && (
              <span className={`text-xs ${riskLevelColor(riskLevel)}`}>{risk}</span>
            )}
            {publishable === false && (
              <span className="rounded-full bg-[rgb(var(--studio-danger-bg))] px-2 py-0.5 text-[10px] text-[rgb(var(--studio-danger-fg))]">
                不建议直接上架
              </span>
            )}
          </div>
          <span className="inline-flex shrink-0 items-center gap-0.5 text-xs text-studio-accent">
            展开报告
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
          ? "flex min-h-0 flex-1 flex-col overflow-hidden bg-gradient-to-br from-studio-ai/5 via-transparent to-studio-panel/40"
          : "shrink-0 border-b border-studio-border bg-gradient-to-br from-studio-ai/5 via-transparent to-studio-panel/40"
      }
    >
      <div className="flex items-center justify-between gap-2 px-4 py-2.5">
        <div className="flex items-center gap-2">
          <Bot className="h-4 w-4 text-studio-ai" />
          <h3 className="text-sm font-medium text-studio-text">AI 味检测</h3>
          {publishable === true && (
            <span className="inline-flex items-center gap-0.5 rounded-full bg-[rgb(var(--studio-diff-add-bg))] px-2 py-0.5 text-[10px] text-[rgb(var(--studio-diff-add-stat))]">
              <ShieldCheck className="h-3 w-3" />
              可尝试上架
            </span>
          )}
          {publishable === false && (
            <span className="inline-flex items-center gap-0.5 rounded-full bg-[rgb(var(--studio-danger-bg))] px-2 py-0.5 text-[10px] text-[rgb(var(--studio-danger-fg))]">
              <AlertTriangle className="h-3 w-3" />
              建议润色后再上架
            </span>
          )}
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

      <div className={`overflow-y-auto px-4 pb-3 ${fill ? "min-h-0 flex-1" : "max-h-[min(280px,36vh)]"}`}>
        <div className="grid gap-2 sm:grid-cols-3">
          <div className="rounded-lg border border-studio-border bg-studio-panel/80 p-2.5">
            <p className="text-[10px] font-medium uppercase tracking-wide text-studio-muted">AI 味</p>
            {aiScore != null ? (
              <p className={`mt-0.5 text-xl font-semibold tabular-nums ${aiScoreColor(aiScore)}`}>
                {aiScore.toFixed(1)}
                <span className="ml-1 text-xs font-normal text-studio-muted">/ 10</span>
              </p>
            ) : (
              <p className="mt-0.5 text-xs text-studio-muted">暂无</p>
            )}
            <p className="mt-1 text-[10px] text-studio-muted">越低越像人写</p>
          </div>

          <div className="rounded-lg border border-studio-border bg-studio-panel/80 p-2.5">
            <p className="text-[10px] font-medium uppercase tracking-wide text-studio-muted">人味</p>
            {humanScore != null ? (
              <p className="mt-0.5 text-xl font-semibold tabular-nums text-studio-text">
                {humanScore.toFixed(1)}
                <span className="ml-1 text-xs font-normal text-studio-muted">/ 10</span>
              </p>
            ) : (
              <p className="mt-0.5 text-xs text-studio-muted">暂无</p>
            )}
          </div>

          <div className="rounded-lg border border-studio-border bg-studio-panel/80 p-2.5">
            <p className="text-[10px] font-medium uppercase tracking-wide text-studio-muted">上架风险</p>
            <p className={`mt-0.5 text-sm font-semibold ${riskLevelColor(riskLevel)}`}>{risk}</p>
          </div>
        </div>

        {signals.length > 0 && (
          <div className="mt-3">
            <p className="mb-1 text-[11px] font-medium text-studio-muted">典型 AI 信号</p>
            <ul className="space-y-1">
              {signals.map((s, i) => (
                <li
                  key={i}
                  className="rounded-md border border-studio-border bg-studio-panel/60 px-2.5 py-1.5 text-xs leading-relaxed text-studio-text"
                >
                  {s}
                </li>
              ))}
            </ul>
          </div>
        )}

        {hotspots.length > 0 && (
          <div className="mt-3">
            <p className="mb-1 text-[11px] font-medium text-studio-muted">高风险片段</p>
            <ul className="space-y-2">
              {hotspots.map((h, i) => (
                <li
                  key={i}
                  className="rounded-md border border-[rgb(var(--studio-warning-border))] bg-[rgb(var(--studio-warning-bg))] px-2.5 py-2 text-xs"
                >
                  {h.excerpt && (
                    <p className="font-medium text-studio-text">「{h.excerpt}」</p>
                  )}
                  {h.reason && <p className="mt-1 text-studio-muted">{h.reason}</p>}
                  {h.fix && (
                    <p className="mt-1 text-[rgb(var(--studio-diff-add-stat))]">建议：{h.fix}</p>
                  )}
                </li>
              ))}
            </ul>
          </div>
        )}

        <p className="mt-3 text-[10px] leading-relaxed text-studio-muted/80">
          仅供参考，不能替代平台审核。切换「完整报告」子标签可查看 Markdown 全文。
        </p>
      </div>
    </section>
  );
}
