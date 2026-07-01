import { useCallback, useEffect, useState } from "react";
import {
  BookOpen,
  Brain,
  ChevronRight,
  Coins,
  FileText,
  GitBranch,
  Loader2,
  PenLine,
  ShieldAlert,
  Sparkles,
} from "lucide-react";
import { ConsistencyReportDTO, ProjectHealthDTO, ProjectTokenUsageDTO, StatusReport, app, formatWordCount, phaseLabel } from "../lib/wails";
import { formatTokenUsage } from "../lib/tokenUsage";
import OnboardingChecklist from "./OnboardingChecklist";
import ProjectHealthPanel from "./ProjectHealthPanel";

type Props = {
  novelId: string;
  status: StatusReport;
  healthRefreshKey: number;
  onContinueWrite: () => void;
  onOpenPlanning: (volume?: number) => void;
  onOpenWrite: (chapter?: number) => void;
  onOpenSettings: () => void;
  onRebuildIndex: () => Promise<void>;
  onReviewChapter: (chapter: number) => void;
  onGoToChapters: () => void;
  onGoToCurrentChapter: () => void;
  onGoToMemories: () => void;
  onGoToForeshadows: () => void;
  onGoToConsistency: () => void;
};

export default function OverviewPanel({
  novelId,
  status,
  healthRefreshKey,
  onContinueWrite,
  onOpenPlanning,
  onOpenWrite,
  onOpenSettings,
  onRebuildIndex,
  onReviewChapter,
  onGoToChapters,
  onGoToCurrentChapter,
  onGoToMemories,
  onGoToForeshadows,
  onGoToConsistency,
}: Props) {
  const nextChapter = Math.max(1, status.current_chapter + 1);
  const pct = Math.min(100, Math.max(0, status.progress_percent ?? 0));
  const hasTarget = (status.target_words ?? 0) > 0;

  return (
    <div className="mx-auto w-full max-w-5xl space-y-6 pb-4">
      <OnboardingChecklist
        novelId={novelId}
        status={status}
        onOpenSettings={onOpenSettings}
        onOpenPlanning={onOpenPlanning}
        onOpenWrite={() => onOpenWrite()}
      />

      <section className="overflow-hidden rounded-2xl border border-studio-border bg-studio-panel shadow-sm">
        <div className="border-b border-studio-border/60 bg-gradient-to-br from-studio-accent/5 to-transparent px-6 py-5">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                <h2 className="text-lg font-semibold text-studio-text">{status.title || "未命名"}</h2>
                <span className="rounded-full bg-studio-accent/10 px-2.5 py-0.5 text-xs text-studio-accent">
                  {phaseLabel[status.phase] || status.phase}
                </span>
              </div>
              <p className="mt-2 text-sm text-studio-muted">
                第 {Math.max(1, status.current_volume)} 卷 · 已写 {status.chapter_count} 章
                {status.current_chapter > 0 && ` · 最新第 ${status.current_chapter} 章`}
              </p>
            </div>
            <div className="text-right">
              <p className="text-3xl font-semibold tabular-nums text-studio-accent">{pct.toFixed(1)}%</p>
              <p className="mt-0.5 text-xs text-studio-muted">创作进度</p>
            </div>
          </div>

          {hasTarget && (
            <div className="mt-4">
              <div className="mb-2 flex flex-wrap items-baseline justify-between gap-2 text-sm">
                <span className="font-medium text-studio-text">
                  {formatWordCount(status.written_words ?? 0)}
                  <span className="mx-1.5 font-normal text-studio-muted">/</span>
                  <span className="text-studio-accent">{formatWordCount(status.target_words ?? 0)}</span>
                </span>
                {status.remaining_words > 0 && (
                  <span className="text-xs text-studio-muted">
                    还差 {formatWordCount(status.remaining_words)}
                    {status.remaining_chapters > 0 && ` · 约 ${status.remaining_chapters} 章`}
                  </span>
                )}
              </div>
              <div className="h-2 overflow-hidden rounded-full bg-studio-bg">
                <div
                  className="h-full rounded-full bg-gradient-to-r from-studio-accent/70 to-studio-accent transition-all duration-500"
                  style={{ width: `${pct}%` }}
                />
              </div>
            </div>
          )}
        </div>

        <div className="flex flex-wrap items-center gap-3 px-6 py-4">
          <button
            type="button"
            onClick={onContinueWrite}
            className="inline-flex items-center gap-2 rounded-xl bg-studio-accent px-4 py-2.5 text-sm font-medium text-studio-on-accent hover:brightness-110"
          >
            <PenLine className="h-4 w-4" />
            继续写第 {nextChapter} 章
          </button>
          <button
            type="button"
            onClick={() => onOpenWrite()}
            className="inline-flex items-center gap-2 rounded-xl border border-studio-border px-4 py-2.5 text-sm text-studio-muted hover:bg-studio-bg hover:text-studio-text"
          >
            打开写作页
          </button>
        </div>
      </section>

      <div className="grid gap-6 lg:grid-cols-5">
        <div className="lg:col-span-3">
          <ProjectHealthPanel
            refreshKey={healthRefreshKey}
            onPlanVolume={(vol) => onOpenPlanning(vol)}
            onOpenWrite={onOpenWrite}
            onRebuildIndex={onRebuildIndex}
            onReviewChapter={onReviewChapter}
            embedded
          />
        </div>
        <div className="space-y-4 lg:col-span-2">
          <ConsistencySummaryCard refreshKey={healthRefreshKey} onGoToConsistency={onGoToConsistency} />
          <TokenUsageCard refreshKey={healthRefreshKey} />
          <OverviewMetrics
            status={status}
            onGoToChapters={onGoToChapters}
            onGoToCurrentChapter={onGoToCurrentChapter}
            onGoToMemories={onGoToMemories}
            onGoToForeshadows={onGoToForeshadows}
          />
          <VolumeOutlineStatus
            volume={Math.max(1, status.current_volume)}
            refreshKey={healthRefreshKey}
            onOpenPlanning={onOpenPlanning}
          />
        </div>
      </div>
    </div>
  );
}

function ConsistencySummaryCard({
  refreshKey,
  onGoToConsistency,
}: {
  refreshKey: number;
  onGoToConsistency: () => void;
}) {
  const [report, setReport] = useState<ConsistencyReportDTO | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const r = await app().GetConsistencyReport();
      setReport(r);
    } catch {
      setReport(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load, refreshKey]);

  const s = report?.summary;
  const hasIssues = (s?.total_issues ?? 0) > 0;

  return (
    <div
      className={`rounded-xl border p-4 ${
        hasIssues
          ? "border-[rgb(var(--studio-warning-border))] bg-[rgb(var(--studio-warning-bg))]"
          : "border-studio-border bg-studio-panel"
      }`}
    >
      <div className="mb-3 flex items-center gap-2">
        <ShieldAlert className={`h-3.5 w-3.5 ${hasIssues ? "text-[rgb(var(--studio-warning-fg))]" : "text-studio-muted"}`} />
        <h3 className="text-xs font-medium uppercase tracking-wide text-studio-muted">一致性</h3>
      </div>
      {loading ? (
        <div className="flex items-center gap-2 text-sm text-studio-muted">
          <Loader2 className="h-4 w-4 animate-spin" />
          检测中…
        </div>
      ) : s ? (
        <>
          <div className="grid grid-cols-2 gap-2 text-sm">
            <div>
              <p className="text-[10px] text-studio-muted">Open 伏笔</p>
              <p className="font-semibold tabular-nums">{s.open_foreshadows}</p>
            </div>
            <div>
              <p className="text-[10px] text-studio-muted">超期伏笔</p>
              <p className={`font-semibold tabular-nums ${s.overdue_foreshadows + s.critical_foreshadows > 0 ? "text-[rgb(var(--studio-warning-fg))]" : ""}`}>
                {s.overdue_foreshadows + s.critical_foreshadows}
              </p>
            </div>
            <div>
              <p className="text-[10px] text-studio-muted">记忆冲突</p>
              <p className={`font-semibold tabular-nums ${s.memory_conflicts > 0 ? "text-[rgb(var(--studio-warning-fg))]" : ""}`}>
                {s.memory_conflicts}
              </p>
            </div>
            <div>
              <p className="text-[10px] text-studio-muted">待处理</p>
              <p className={`font-semibold tabular-nums ${hasIssues ? "text-[rgb(var(--studio-warning-fg))]" : "text-[rgb(var(--studio-diff-add-stat))]"}`}>
                {s.total_issues}
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={onGoToConsistency}
            className="mt-4 inline-flex w-full items-center justify-center gap-1.5 rounded-lg border border-studio-border px-3 py-2 text-sm text-studio-text transition hover:bg-studio-bg"
          >
            {hasIssues ? "查看并处理" : "打开一致性仪表盘"}
            <ChevronRight className="h-4 w-4 text-studio-muted" />
          </button>
        </>
      ) : null}
    </div>
  );
}

function TokenUsageCard({ refreshKey }: { refreshKey: number }) {
  const [usage, setUsage] = useState<ProjectTokenUsageDTO | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const u = await app().GetProjectTokenUsage();
      setUsage(u);
    } catch {
      setUsage(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load, refreshKey]);

  if (loading) return null;
  if (!usage || usage.total_tokens <= 0) return null;

  return (
    <div className="rounded-xl border border-studio-border bg-studio-panel p-4">
      <div className="mb-2 flex items-center gap-2">
        <Coins className="h-3.5 w-3.5 text-studio-muted" />
        <h3 className="text-xs font-medium uppercase tracking-wide text-studio-muted">API 用量累计</h3>
      </div>
      <p className="text-sm font-medium text-studio-text">{formatTokenUsage(usage)}</p>
      <p className="mt-1 text-[11px] text-studio-muted">写章流水线 token 粗估，成本按当前模型档位估算</p>
    </div>
  );
}

function OverviewMetrics({
  status,
  onGoToChapters,
  onGoToCurrentChapter,
  onGoToMemories,
  onGoToForeshadows,
}: {
  status: StatusReport;
  onGoToChapters: () => void;
  onGoToCurrentChapter: () => void;
  onGoToMemories: () => void;
  onGoToForeshadows: () => void;
}) {
  const items = [
    {
      label: "章节",
      value: status.chapter_count,
      icon: FileText,
      onClick: status.chapter_count > 0 ? onGoToChapters : undefined,
    },
    {
      label: "当前章",
      value: status.current_chapter,
      icon: BookOpen,
      onClick: status.current_chapter > 0 ? onGoToCurrentChapter : undefined,
    },
    {
      label: "记忆",
      value: status.memory_count,
      icon: Brain,
      onClick: status.memory_count > 0 ? onGoToMemories : undefined,
    },
    {
      label: "Open 伏笔",
      value: status.open_foreshadows,
      icon: GitBranch,
      onClick: status.open_foreshadows > 0 ? onGoToForeshadows : undefined,
    },
  ];

  return (
    <div className="rounded-xl border border-studio-border bg-studio-panel p-4">
      <h3 className="mb-3 text-xs font-medium uppercase tracking-wide text-studio-muted">数据一览</h3>
      <div className="grid grid-cols-2 gap-2">
        {items.map(({ label, value, icon: Icon, onClick }) => {
          const clickable = !!onClick && value > 0;
          const inner = (
            <>
              <Icon className="h-3.5 w-3.5 shrink-0 text-studio-muted" />
              <div className="min-w-0 flex-1">
                <p className="text-[10px] uppercase tracking-wide text-studio-muted">{label}</p>
                <p className="text-lg font-semibold tabular-nums leading-tight">{value}</p>
              </div>
            </>
          );
          if (clickable) {
            return (
              <button
                key={label}
                type="button"
                onClick={onClick}
                className="group flex items-center gap-2 rounded-lg border border-transparent bg-studio-bg/60 p-2.5 text-left transition hover:border-studio-border hover:bg-studio-bg"
              >
                {inner}
              </button>
            );
          }
          return (
            <div
              key={label}
              className="flex items-center gap-2 rounded-lg bg-studio-bg/40 p-2.5 opacity-80"
            >
              {inner}
            </div>
          );
        })}
      </div>
    </div>
  );
}

function VolumeOutlineStatus({
  volume,
  refreshKey,
  onOpenPlanning,
}: {
  volume: number;
  refreshKey: number;
  onOpenPlanning: (volume?: number) => void;
}) {
  const [health, setHealth] = useState<ProjectHealthDTO | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const h = await app().GetProjectHealth();
      setHealth(h);
    } catch {
      setHealth(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load, refreshKey]);

  const ready = health?.has_volume_outline ?? false;

  return (
    <div className="rounded-xl border border-studio-border bg-studio-panel p-4">
      <h3 className="mb-3 text-xs font-medium uppercase tracking-wide text-studio-muted">卷纲</h3>
      {loading ? (
        <div className="flex items-center gap-2 text-sm text-studio-muted">
          <Loader2 className="h-4 w-4 animate-spin" />
          加载中…
        </div>
      ) : (
        <>
          <div className="flex items-start gap-3">
            <div
              className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-lg ${
                ready ? "bg-[rgb(var(--studio-diff-add-bg))]" : "bg-[rgb(var(--studio-warning-bg))]"
              }`}
            >
              <Sparkles
                className={`h-4 w-4 ${
                  ready ? "text-[rgb(var(--studio-diff-add-stat))]" : "text-[rgb(var(--studio-warning-fg))]"
                }`}
              />
            </div>
            <div className="min-w-0 flex-1">
              <p className="text-sm font-medium">第 {volume} 卷</p>
              <p className="mt-0.5 text-xs text-studio-muted">
                {ready ? "卷纲已就绪，写章时可提取章纲" : "尚未生成卷纲，写章前建议先规划"}
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={() => onOpenPlanning(volume)}
            className="mt-4 inline-flex w-full items-center justify-center gap-1.5 rounded-lg border border-studio-border px-3 py-2 text-sm text-studio-text transition hover:bg-studio-bg"
          >
            {ready ? "查看 / 编辑卷纲" : "去生成卷纲"}
            <ChevronRight className="h-4 w-4 text-studio-muted" />
          </button>
        </>
      )}
    </div>
  );
}
