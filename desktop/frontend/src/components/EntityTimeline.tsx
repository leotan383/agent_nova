import { History, Loader2 } from "lucide-react";
import { EntityStateSnapshotDTO } from "../lib/wails";

function skipField(key: string) {
  return key === "aliases";
}

function diffFromPrevious(
  prev: Record<string, string> | undefined,
  curr: Record<string, string>,
) {
  const changed = new Set<string>();
  const added = new Set<string>();
  if (!prev) {
    for (const k of Object.keys(curr)) {
      if (!skipField(k)) added.add(k);
    }
    return { changed, added };
  }
  for (const [k, v] of Object.entries(curr)) {
    if (skipField(k)) continue;
    if (!(k in prev)) added.add(k);
    else if (prev[k] !== v) changed.add(k);
  }
  return { changed, added };
}

type Props = {
  snapshots: EntityStateSnapshotDTO[];
  loading: boolean;
  accentClass: string;
  lastChapter: number;
  needsBackfill: boolean;
  backfillRunning: boolean;
  backfillMessage: string;
  backfillError: string;
  onBackfill: () => void;
};

export default function EntityTimeline({
  snapshots,
  loading,
  accentClass,
  lastChapter,
  needsBackfill,
  backfillRunning,
  backfillMessage,
  backfillError,
  onBackfill,
}: Props) {
  if (loading) {
    return (
      <div className="flex items-center justify-center py-16 text-studio-muted">
        <Loader2 className="h-6 w-6 animate-spin" />
      </div>
    );
  }

  if (snapshots.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center">
        <p className="text-sm text-studio-muted">暂无历史记录</p>
        <p className="mt-1 max-w-sm text-xs leading-relaxed text-studio-muted/70">
          从下一章写章审查起，每次 AI 提取都会在此追加一条快照。
        </p>
        {lastChapter > 0 && (
          <BackfillBanner
            needsBackfill
            backfillRunning={backfillRunning}
            backfillMessage={backfillMessage}
            backfillError={backfillError}
            onBackfill={onBackfill}
            className="mt-6 max-w-md"
          />
        )}
      </div>
    );
  }

  return (
    <div className="relative mx-auto max-w-2xl py-2">
      {needsBackfill && (
        <BackfillBanner
          needsBackfill={needsBackfill}
          backfillRunning={backfillRunning}
          backfillMessage={backfillMessage}
          backfillError={backfillError}
          onBackfill={onBackfill}
          className="mb-6"
        />
      )}

      <div
        className="absolute bottom-4 left-[1.125rem] top-4 w-px bg-gradient-to-b from-transparent via-studio-border to-transparent"
        aria-hidden
      />

      <ol className="space-y-0">
        {snapshots.map((snap, index) => {
          const prev = index > 0 ? snapshots[index - 1].state : undefined;
          const { changed, added } = diffFromPrevious(prev, snap.state);
          const fields = Object.entries(snap.state).filter(([k]) => !skipField(k));
          const isLast = index === snapshots.length - 1;

          return (
            <li key={`${snap.chapter}-${index}`} className="relative pl-12 pb-10 last:pb-4">
              <div
                className={`absolute left-0 top-1 flex h-9 w-9 items-center justify-center rounded-full border-2 text-[11px] font-semibold tabular-nums ${
                  snap.is_current
                    ? `${accentClass} border-current bg-studio-panel shadow-sm`
                    : "border-studio-border bg-studio-bg text-studio-muted"
                }`}
              >
                {snap.chapter}
              </div>

              <div
                className={`rounded-xl border bg-studio-bg/40 transition ${
                  snap.is_current
                    ? "border-studio-accent/30 shadow-sm shadow-studio-accent/5"
                    : "border-studio-border"
                }`}
              >
                <div className="flex flex-wrap items-baseline justify-between gap-2 border-b border-studio-border/60 px-4 py-3">
                  <div>
                    <p className="text-sm font-medium text-studio-text">
                      第 {snap.chapter} 章
                      {snap.chapter_title ? (
                        <span className="ml-1.5 font-normal text-studio-muted">· {snap.chapter_title}</span>
                      ) : null}
                    </p>
                    {snap.recorded_at && (
                      <p className="mt-0.5 text-[10px] text-studio-muted/80">
                        {formatRecordedAt(snap.recorded_at)}
                      </p>
                    )}
                  </div>
                  {snap.is_current && (
                    <span className="rounded-full bg-studio-accent/10 px-2 py-0.5 text-[10px] font-medium text-studio-accent">
                      当前
                    </span>
                  )}
                  {!snap.is_current && isLast && snapshots.length > 1 && (
                    <span className="rounded-full bg-studio-bg px-2 py-0.5 text-[10px] text-studio-muted">
                      最新快照
                    </span>
                  )}
                </div>

                {fields.length > 0 ? (
                  <dl className="grid gap-2 p-4 sm:grid-cols-2">
                    {fields.map(([k, v]) => {
                      const isChanged = changed.has(k);
                      const isAdded = added.has(k);
                      return (
                        <div
                          key={k}
                          className={`rounded-lg px-3 py-2.5 ${
                            isChanged
                              ? "bg-studio-accent/5 ring-1 ring-studio-accent/20"
                              : isAdded
                                ? "bg-[rgb(var(--studio-diff-add-bg))]/40 ring-1 ring-[rgb(var(--studio-diff-add-stat))]/15"
                                : "bg-studio-panel/50"
                          }`}
                        >
                          <dt className="flex items-center gap-1.5 text-[10px] font-medium uppercase tracking-wide text-studio-muted">
                            {k}
                            {isChanged && (
                              <span className="normal-case text-studio-accent">已变化</span>
                            )}
                            {isAdded && !isChanged && (
                              <span className="normal-case text-[rgb(var(--studio-diff-add-stat))]">新增</span>
                            )}
                          </dt>
                          <dd className="mt-1 text-sm leading-relaxed text-studio-text">{v}</dd>
                        </div>
                      );
                    })}
                  </dl>
                ) : (
                  <p className="px-4 py-5 text-sm text-studio-muted">本章未提取到结构化字段</p>
                )}
              </div>
            </li>
          );
        })}
      </ol>
    </div>
  );
}

function BackfillBanner({
  needsBackfill,
  backfillRunning,
  backfillMessage,
  backfillError,
  onBackfill,
  className = "",
}: {
  needsBackfill: boolean;
  backfillRunning: boolean;
  backfillMessage: string;
  backfillError: string;
  onBackfill: () => void;
  className?: string;
}) {
  if (!needsBackfill && !backfillRunning && !backfillError) {
    return null;
  }

  return (
    <div
      className={`rounded-xl border border-studio-border/80 bg-studio-panel/60 px-4 py-3 text-left ${className}`}
    >
      <p className="text-sm text-studio-text">早期章节的状态尚未记录</p>
      <p className="mt-1 text-xs leading-relaxed text-studio-muted">
        状态时间线是在近期功能上线后才开始逐章保存的，因此只显示最新一次审查的快照。可从已写章节重新提取，补齐各章状态（会调用
        AI，按章节数计费）。
      </p>
      {backfillMessage && (
        <p className="mt-2 flex items-center gap-1.5 text-xs text-studio-muted">
          {backfillRunning && <Loader2 className="h-3.5 w-3.5 animate-spin shrink-0" />}
          {backfillMessage}
        </p>
      )}
      {backfillError && <p className="mt-2 text-xs text-red-500">{backfillError}</p>}
      <button
        type="button"
        disabled={backfillRunning}
        onClick={onBackfill}
        className="mt-3 inline-flex items-center gap-1.5 rounded-lg bg-studio-accent/15 px-3 py-1.5 text-xs font-medium text-studio-accent transition hover:bg-studio-accent/25 disabled:opacity-50"
      >
        {backfillRunning ? (
          <>
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
            回溯中…
          </>
        ) : (
          <>
            <History className="h-3.5 w-3.5" />
            从历史章节回溯填充
          </>
        )}
      </button>
    </div>
  );
}

function formatRecordedAt(iso: string) {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleString("zh-CN", {
    month: "numeric",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}
