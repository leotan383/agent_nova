import { useCallback, useEffect, useState } from "react";
import {
  BookOpen,
  Brain,
  ChevronDown,
  ChevronRight,
  Coins,
  FileText,
  GitBranch,
  Loader2,
  PenLine,
  ShieldCheck,
  ShieldAlert,
  Sparkles,
  Wrench,
  type LucideIcon,
} from "lucide-react";
import {
  ConsistencyReportDTO,
  ProjectHealthDTO,
  ProjectTokenUsageDTO,
  StatusReport,
  app,
  formatWordCount,
  phaseLabel,
} from "../lib/wails";
import { formatTokenUsage } from "../lib/tokenUsage";
import OnboardingChecklist from "./OnboardingChecklist";
import DailyWorkflowCard from "./DailyWorkflowCard";
import ProjectHealthPanel from "./ProjectHealthPanel";
import ProjectToolsCard from "./ProjectToolsCard";

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
  onGoToMemoryConflicts: () => void;
  onProjectToolsRefresh?: () => void;
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
  onGoToMemoryConflicts,
  onProjectToolsRefresh,
}: Props) {
  const nextChapter = Math.max(1, status.current_chapter + 1);
  const pct = Math.min(100, Math.max(0, status.progress_percent ?? 0));
  const hasTarget = (status.target_words ?? 0) > 0;
  const volume = Math.max(1, status.current_volume);

  return (
    <div className="mx-auto w-full max-w-4xl space-y-5 pb-8">
      <OnboardingChecklist
        novelId={novelId}
        status={status}
        onOpenSettings={onOpenSettings}
        onOpenPlanning={onOpenPlanning}
        onOpenWrite={() => onOpenWrite()}
      />

      <section className="relative overflow-hidden rounded-2xl border border-studio-border bg-studio-panel shadow-sm">
        <div className="pointer-events-none absolute inset-0 bg-gradient-to-br from-studio-accent/[0.06] via-transparent to-transparent" />
        <div className="relative px-6 py-6">
          <div className="flex flex-wrap items-start justify-between gap-5">
            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-center gap-2">
                <h2 className="text-xl font-semibold tracking-tight text-studio-text">
                  {status.title || "未命名"}
                </h2>
                <span className="rounded-full bg-studio-accent/10 px-2.5 py-0.5 text-xs font-medium text-studio-accent">
                  {phaseLabel[status.phase] || status.phase}
                </span>
              </div>
              <p className="mt-1.5 text-sm text-studio-muted">
                第 {volume} 卷 · {status.chapter_count} 章
                {status.current_chapter > 0 && ` · 最新第 ${status.current_chapter} 章`}
              </p>
              {status.synopsis?.trim() && (
                <p className="mt-3 line-clamp-2 max-w-xl text-sm leading-relaxed text-studio-muted/90">
                  {status.synopsis.trim()}
                </p>
              )}
            </div>

            <div className="flex shrink-0 flex-col items-end">
              <span className="text-3xl font-semibold tabular-nums leading-none text-studio-accent">
                {pct.toFixed(0)}%
              </span>
              <span className="mt-1 text-xs text-studio-muted">总进度</span>
            </div>
          </div>

          {hasTarget && (
            <div className="mt-5">
              <div className="mb-2 flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1 text-sm">
                <span className="text-studio-muted">
                  <span className="font-medium text-studio-text">
                    {formatWordCount(status.written_words ?? 0)}
                  </span>
                  <span className="mx-1.5">/</span>
                  {formatWordCount(status.target_words ?? 0)}
                </span>
                {status.remaining_words > 0 && (
                  <span className="text-xs text-studio-muted">
                    还差 {formatWordCount(status.remaining_words)}
                    {status.remaining_chapters > 0 && ` · 约 ${status.remaining_chapters} 章`}
                  </span>
                )}
              </div>
              <div className="h-1.5 overflow-hidden rounded-full bg-studio-bg">
                <div
                  className="h-full rounded-full bg-studio-accent transition-all duration-500"
                  style={{ width: `${pct}%` }}
                />
              </div>
            </div>
          )}

          <div className="mt-5 grid grid-cols-2 gap-2 sm:grid-cols-4">
            <HeroStat
              icon={FileText}
              label="章节"
              value={status.chapter_count}
              onClick={status.chapter_count > 0 ? onGoToChapters : undefined}
            />
            <HeroStat
              icon={BookOpen}
              label="当前章"
              value={status.current_chapter}
              onClick={status.current_chapter > 0 ? onGoToCurrentChapter : undefined}
            />
            <HeroStat
              icon={Brain}
              label="记忆"
              value={status.memory_count}
              onClick={status.memory_count > 0 ? onGoToMemories : undefined}
            />
            <HeroStat
              icon={GitBranch}
              label="Open 伏笔"
              value={status.open_foreshadows}
              onClick={status.open_foreshadows > 0 ? onGoToForeshadows : undefined}
            />
          </div>

          <div className="mt-5 flex flex-wrap items-center gap-3 border-t border-studio-border/50 pt-5">
            <button
              type="button"
              onClick={onContinueWrite}
              className="inline-flex items-center gap-2 rounded-xl bg-studio-accent px-5 py-2.5 text-sm font-medium text-studio-on-accent shadow-sm transition hover:brightness-110"
            >
              <PenLine className="h-4 w-4" />
              继续写第 {nextChapter} 章
            </button>
            <button
              type="button"
              onClick={() => onOpenWrite()}
              className="text-sm text-studio-muted transition hover:text-studio-text"
            >
              打开写作页
            </button>
          </div>
        </div>
      </section>

      <div className="grid gap-5 lg:grid-cols-12">
        <div className="flex flex-col gap-5 lg:col-span-7">
          <DailyWorkflowCard
            refreshKey={healthRefreshKey}
            onOpenWrite={() => onOpenWrite()}
            onReviewChapter={onReviewChapter}
            onOpenChapters={onGoToChapters}
          />
          <ProjectHealthPanel
            refreshKey={healthRefreshKey}
            onPlanVolume={(vol) => onOpenPlanning(vol)}
            onOpenWrite={onOpenWrite}
            onRebuildIndex={onRebuildIndex}
            onReviewChapter={onReviewChapter}
            embedded
          />
        </div>

        <aside className="flex flex-col gap-4 lg:col-span-5">
          <ProjectStatusCard
            volume={volume}
            refreshKey={healthRefreshKey}
            onOpenPlanning={onOpenPlanning}
            onGoToMemoryConflicts={onGoToMemoryConflicts}
            onGoToForeshadows={onGoToForeshadows}
            onGoToMemories={onGoToMemories}
          />

          <details className="group rounded-2xl border border-studio-border bg-studio-panel">
            <summary className="flex cursor-pointer list-none items-center justify-between px-4 py-3.5 text-sm text-studio-muted transition hover:text-studio-text [&::-webkit-details-marker]:hidden">
              <span className="flex items-center gap-2">
                <Wrench className="h-4 w-4" />
                项目工具与诊断
              </span>
              <ChevronDown className="h-4 w-4 shrink-0 transition group-open:rotate-180" />
            </summary>
            <div className="space-y-3 border-t border-studio-border/60 px-4 pb-4 pt-3">
              <ProjectToolsCard
                embedded
                refreshKey={healthRefreshKey}
                onRefresh={onProjectToolsRefresh}
              />
              <TokenUsageInline refreshKey={healthRefreshKey} />
            </div>
          </details>
        </aside>
      </div>
    </div>
  );
}

function HeroStat({
  icon: Icon,
  label,
  value,
  onClick,
}: {
  icon: LucideIcon;
  label: string;
  value: number;
  onClick?: () => void;
}) {
  const clickable = !!onClick && value > 0;
  const className =
    "flex min-w-0 items-center gap-2.5 rounded-xl border border-studio-border/50 bg-studio-bg/30 px-3 py-2.5 transition";
  const content = (
    <>
      <Icon className="h-4 w-4 shrink-0 text-studio-muted" />
      <div className="min-w-0">
        <p className="truncate text-[11px] text-studio-muted">{label}</p>
        <p className="text-base font-semibold tabular-nums leading-tight">{value}</p>
      </div>
    </>
  );

  if (clickable) {
    return (
      <button
        type="button"
        onClick={onClick}
        className={`${className} text-left hover:border-studio-border hover:bg-studio-bg/60`}
      >
        {content}
      </button>
    );
  }

  return <div className={`${className} opacity-75`}>{content}</div>;
}

function ProjectStatusCard({
  volume,
  refreshKey,
  onOpenPlanning,
  onGoToMemoryConflicts,
  onGoToForeshadows,
  onGoToMemories,
}: {
  volume: number;
  refreshKey: number;
  onOpenPlanning: (volume?: number) => void;
  onGoToMemoryConflicts: () => void;
  onGoToForeshadows: () => void;
  onGoToMemories: () => void;
}) {
  const [health, setHealth] = useState<ProjectHealthDTO | null>(null);
  const [report, setReport] = useState<ConsistencyReportDTO | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [h, r] = await Promise.all([app().GetProjectHealth(), app().GetConsistencyReport()]);
      setHealth(h);
      setReport(r);
    } catch {
      setHealth(null);
      setReport(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load, refreshKey]);

  const ready = health?.has_volume_outline ?? false;
  const s = report?.summary;
  const hasIssues = (s?.total_issues ?? 0) > 0;
  const overdueForeshadows = (s?.overdue_foreshadows ?? 0) + (s?.critical_foreshadows ?? 0);

  const handleSerialAction = () => {
    if (!s) return;
    if (s.memory_conflicts > 0) {
      onGoToMemoryConflicts();
      return;
    }
    if (overdueForeshadows > 0 || s.open_foreshadows > 0) {
      onGoToForeshadows();
      return;
    }
    onGoToMemories();
  };

  const serialActionLabel = (() => {
    if (!s || !hasIssues) return "";
    if (s.memory_conflicts > 0) return "处理记忆冲突";
    if (overdueForeshadows > 0) return "查看超期伏笔";
    if (s.open_foreshadows > 0) return "查看 Open 伏笔";
    return "查看记忆";
  })();

  return (
    <div className="overflow-hidden rounded-2xl border border-studio-border bg-studio-panel shadow-sm">
      <div className="border-b border-studio-border/60 px-4 py-3">
        <h3 className="text-sm font-medium text-studio-text">项目状态</h3>
      </div>

      {loading ? (
        <div className="flex items-center gap-2 px-4 py-6 text-sm text-studio-muted">
          <Loader2 className="h-4 w-4 animate-spin" />
          加载中…
        </div>
      ) : (
        <div className="divide-y divide-studio-border/50">
          <div className="flex items-center gap-3 px-4 py-3.5">
            <div
              className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-xl ${
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
              <p className="text-sm font-medium text-studio-text">第 {volume} 卷卷纲</p>
              <p className="mt-0.5 text-xs text-studio-muted">
                {ready ? "已就绪，写章时可提取章纲" : "建议写章前先规划卷纲"}
              </p>
            </div>
            <button
              type="button"
              onClick={() => onOpenPlanning(volume)}
              className="shrink-0 rounded-lg border border-studio-border px-2.5 py-1.5 text-xs text-studio-text transition hover:bg-studio-bg"
            >
              {ready ? "查看" : "去规划"}
            </button>
          </div>

          <div className="px-4 py-3.5">
            <div className="mb-3 flex items-center gap-2">
              {hasIssues ? (
                <ShieldAlert className="h-4 w-4 text-[rgb(var(--studio-warning-fg))]" />
              ) : (
                <ShieldCheck className="h-4 w-4 text-[rgb(var(--studio-diff-add-stat))]" />
              )}
              <span className="text-sm font-medium text-studio-text">连载检查</span>
              {!hasIssues && s && (
                <span className="ml-auto text-xs text-[rgb(var(--studio-diff-add-stat))]">一切正常</span>
              )}
            </div>

            {s ? (
              <>
                <div className="flex gap-2">
                  <StatusPill
                    label="Open 伏笔"
                    value={s.open_foreshadows}
                    warn={s.open_foreshadows > 0 && hasIssues}
                  />
                  <StatusPill
                    label="超期"
                    value={overdueForeshadows}
                    warn={overdueForeshadows > 0}
                  />
                  <StatusPill
                    label="记忆冲突"
                    value={s.memory_conflicts}
                    warn={s.memory_conflicts > 0}
                  />
                </div>
                {hasIssues && serialActionLabel && (
                  <button
                    type="button"
                    onClick={handleSerialAction}
                    className="mt-3 inline-flex w-full items-center justify-center gap-1 rounded-lg border border-studio-border px-3 py-2 text-xs text-studio-text transition hover:bg-studio-bg"
                  >
                    {serialActionLabel}
                    <ChevronRight className="h-3.5 w-3.5 text-studio-muted" />
                  </button>
                )}
              </>
            ) : null}
          </div>
        </div>
      )}
    </div>
  );
}

function StatusPill({ label, value, warn }: { label: string; value: number; warn?: boolean }) {
  return (
    <div
      className={`flex min-w-0 flex-1 flex-col items-center rounded-lg px-2 py-2 ${
        warn ? "bg-[rgb(var(--studio-warning-bg))]" : "bg-studio-bg/50"
      }`}
    >
      <span className="truncate text-[10px] text-studio-muted">{label}</span>
      <span
        className={`mt-0.5 text-base font-semibold tabular-nums ${
          warn ? "text-[rgb(var(--studio-warning-fg))]" : "text-studio-text"
        }`}
      >
        {value}
      </span>
    </div>
  );
}

function TokenUsageInline({ refreshKey }: { refreshKey: number }) {
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

  if (loading || !usage || usage.total_tokens <= 0) return null;

  return (
    <div className="flex items-start gap-2 rounded-lg bg-studio-bg/40 px-3 py-2.5">
      <Coins className="mt-0.5 h-3.5 w-3.5 shrink-0 text-studio-muted" />
      <div>
        <p className="text-xs font-medium text-studio-text">{formatTokenUsage(usage)}</p>
        <p className="mt-0.5 text-[11px] text-studio-muted">写章流水线 token 累计</p>
      </div>
    </div>
  );
}
