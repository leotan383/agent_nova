import { Loader2 } from "lucide-react";
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
};

export default function EntityTimeline({ snapshots, loading, accentClass }: Props) {
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
          从下一章写章审查起，每次 AI 提取都会在此追加一条快照。已有角色会显示最近一次的状态。
        </p>
      </div>
    );
  }

  return (
    <div className="relative mx-auto max-w-2xl py-2">
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
