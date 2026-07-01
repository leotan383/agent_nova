import { useCallback, useEffect, useMemo, useState } from "react";
import { ChevronDown, ChevronRight, Loader2, Pin, PinOff, RefreshCw, X } from "lucide-react";
import { WriteContextDTO, app } from "../lib/wails";
import ContentPreview from "./ContentPreview";

type Props = {
  chapter: number;
  volume: number;
  defaultCollapsed?: boolean;
  pinnedIds?: string[];
  excludedIds?: string[];
  onPinnedChange?: (ids: string[]) => void;
  onExcludedChange?: (ids: string[]) => void;
};

function Section({ title, body, defaultOpen = false }: { title: string; body: string; defaultOpen?: boolean }) {
  const [open, setOpen] = useState(defaultOpen);
  if (!body.trim()) return null;
  return (
    <div className="border-b border-studio-border/60 last:border-0">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center gap-2 px-3 py-2 text-left text-[11px] font-medium text-studio-muted hover:bg-studio-bg hover:text-studio-text"
      >
        {open ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
        {title}
      </button>
      {open && (
        <div className="max-h-36 overflow-y-auto border-t border-studio-border/40 px-3 py-2">
          <ContentPreview content={body} className="text-[11px]" />
        </div>
      )}
    </div>
  );
}

const sourceLabels: Record<string, string> = {
  rule: "规则",
  semantic: "语义",
  rrf: "融合",
  fallback: "近期",
  pinned: "固定",
};

function MemoryRecallSection({
  recalls,
  pinnedIds,
  excludedIds,
  onPin,
  onExclude,
}: {
  recalls: WriteContextDTO["memory_recalls"];
  pinnedIds: string[];
  excludedIds: string[];
  onPin: (id: string) => void;
  onExclude: (id: string) => void;
}) {
  const [open, setOpen] = useState(true);
  const visible = recalls?.filter((r) => !excludedIds.includes(r.id)) ?? [];
  if (visible.length === 0 && excludedIds.length === 0) return null;

  return (
    <div className="border-b border-studio-border/60 last:border-0">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center gap-2 px-3 py-2 text-left text-[11px] font-medium text-studio-muted hover:bg-studio-bg hover:text-studio-text"
      >
        {open ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
        长期记忆（{visible.length} 条注入
        {excludedIds.length > 0 ? `，${excludedIds.length} 条已排除` : ""}）
      </button>
      {open && (
        <div className="max-h-52 space-y-2 overflow-y-auto border-t border-studio-border/40 px-3 py-2">
          {visible.map((r) => {
            const pinned = pinnedIds.includes(r.id) || r.source === "pinned";
            return (
              <div
                key={r.id}
                className={`rounded-lg border px-2.5 py-2 ${
                  pinned
                    ? "border-studio-accent/40 bg-studio-accent/5"
                    : "border-studio-border/50 bg-studio-bg/40"
                }`}
              >
                <div className="flex flex-wrap items-center gap-1.5">
                  <span className="text-[11px] font-medium text-studio-text">
                    [{r.category}/{r.subject}]
                  </span>
                  <span className="rounded bg-studio-panel px-1.5 py-0.5 text-[9px] text-studio-muted">
                    {sourceLabels[r.source] ?? r.source}
                  </span>
                  <div className="ml-auto flex gap-0.5">
                    <button
                      type="button"
                      onClick={() => onPin(r.id)}
                      className="rounded p-1 text-studio-muted hover:text-studio-accent"
                      title={pinned ? "取消固定" : "固定注入"}
                    >
                      {pinned ? <PinOff className="h-3 w-3" /> : <Pin className="h-3 w-3" />}
                    </button>
                    <button
                      type="button"
                      onClick={() => onExclude(r.id)}
                      className="rounded p-1 text-studio-muted hover:text-[rgb(var(--studio-warning-fg))]"
                      title="排除本条"
                    >
                      <X className="h-3 w-3" />
                    </button>
                  </div>
                </div>
                <p className="mt-1 text-[10px] text-studio-muted">{r.reason}</p>
                <p className="mt-1 line-clamp-3 text-[11px] text-studio-text/85">{r.content}</p>
              </div>
            );
          })}
          {excludedIds.length > 0 && (
            <p className="text-[10px] text-studio-muted">
              已排除 {excludedIds.length} 条，写章时不会注入
            </p>
          )}
        </div>
      )}
    </div>
  );
}

export default function WriteContextPanel({
  chapter,
  volume,
  defaultCollapsed = true,
  pinnedIds = [],
  excludedIds = [],
  onPinnedChange,
  onExcludedChange,
}: Props) {
  const [ctx, setCtx] = useState<WriteContextDTO | null>(null);
  const [loading, setLoading] = useState(false);
  const [collapsed, setCollapsed] = useState(defaultCollapsed);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await app().GetWriteContext({
        chapter,
        volume,
        pinned_memory_ids: pinnedIds,
        excluded_memory_ids: excludedIds,
      });
      setCtx(data);
    } catch {
      setCtx(null);
    } finally {
      setLoading(false);
    }
  }, [chapter, volume, pinnedIds, excludedIds]);

  useEffect(() => {
    setCollapsed(defaultCollapsed);
    load();
  }, [load, defaultCollapsed]);

  const handlePin = (id: string) => {
    if (!onPinnedChange) return;
    const nextPinned = pinnedIds.includes(id)
      ? pinnedIds.filter((x) => x !== id)
      : [...pinnedIds, id];
    onPinnedChange(nextPinned);
    if (onExcludedChange && excludedIds.includes(id)) {
      onExcludedChange(excludedIds.filter((x) => x !== id));
    }
  };

  const handleExclude = (id: string) => {
    if (!onExcludedChange) return;
    onExcludedChange([...excludedIds, id]);
    if (onPinnedChange && pinnedIds.includes(id)) {
      onPinnedChange(pinnedIds.filter((x) => x !== id));
    }
  };

  const preview = useMemo(() => {
    const outline = ctx?.outline?.trim() || "";
    if (!outline) return "暂无章纲预览";
    const line = outline.split("\n").find((l) => l.trim())?.trim() || outline;
    return line.length > 72 ? `${line.slice(0, 72)}…` : line;
  }, [ctx?.outline]);

  if (collapsed) {
    return (
      <button
        type="button"
        onClick={() => setCollapsed(false)}
        className="w-full rounded-xl border border-studio-border bg-studio-panel px-3 py-2.5 text-left transition hover:border-studio-muted/50"
      >
        <div className="flex items-center justify-between gap-2">
          <span className="text-[10px] font-medium uppercase tracking-wide text-studio-muted">
            本章上下文
          </span>
          <ChevronRight className="h-3.5 w-3.5 text-studio-muted" />
        </div>
        <p className="mt-1 line-clamp-2 text-xs text-studio-text/80">{preview}</p>
      </button>
    );
  }

  return (
    <div className="flex max-h-72 shrink-0 flex-col overflow-hidden rounded-xl border border-studio-border bg-studio-panel">
      <div className="flex shrink-0 items-center justify-between border-b border-studio-border px-3 py-2">
        <span className="text-[10px] font-medium uppercase tracking-wide text-studio-muted">
          第 {chapter} 章上下文
        </span>
        <div className="flex items-center gap-1">
          <button
            type="button"
            onClick={load}
            className="rounded p-1 text-studio-muted hover:text-studio-text"
            title="刷新"
          >
            <RefreshCw className={`h-3.5 w-3.5 ${loading ? "animate-spin" : ""}`} />
          </button>
          <button
            type="button"
            onClick={() => setCollapsed(true)}
            className="rounded px-2 py-0.5 text-[10px] text-studio-muted hover:bg-studio-bg"
          >
            收起
          </button>
        </div>
      </div>
      {loading && !ctx ? (
        <div className="flex items-center justify-center py-6 text-studio-muted">
          <Loader2 className="h-4 w-4 animate-spin" />
        </div>
      ) : (
        <div className="min-h-0 flex-1 overflow-y-auto">
          <Section title="卷纲 / 章纲" body={ctx?.outline ?? ""} defaultOpen />
          <Section title="近章摘要" body={ctx?.recent_summary ?? ""} />
          <Section title="设定摘要" body={ctx?.settings ?? ""} />
          <MemoryRecallSection
            recalls={ctx?.memory_recalls ?? []}
            pinnedIds={pinnedIds}
            excludedIds={excludedIds}
            onPin={handlePin}
            onExclude={handleExclude}
          />
          {!ctx?.memory_recalls?.length && (
            <Section title="长期记忆" body={ctx?.memories ?? ""} />
          )}
          <Section title="Open 伏笔" body={ctx?.open_foreshadows ?? ""} />
          <Section title="检索命中" body={ctx?.fts_hits ?? ""} />
        </div>
      )}
    </div>
  );
}
