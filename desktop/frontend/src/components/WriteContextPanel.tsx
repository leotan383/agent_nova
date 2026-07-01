import { useCallback, useEffect, useMemo, useState } from "react";
import { ChevronDown, ChevronRight, Loader2, RefreshCw } from "lucide-react";
import { WriteContextDTO, app } from "../lib/wails";
import ContentPreview from "./ContentPreview";

type Props = {
  chapter: number;
  volume: number;
  defaultCollapsed?: boolean;
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
};

function MemoryRecallSection({ recalls }: { recalls: WriteContextDTO["memory_recalls"] }) {
  const [open, setOpen] = useState(true);
  if (!recalls?.length) return null;
  return (
    <div className="border-b border-studio-border/60 last:border-0">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center gap-2 px-3 py-2 text-left text-[11px] font-medium text-studio-muted hover:bg-studio-bg hover:text-studio-text"
      >
        {open ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
        长期记忆（{recalls.length} 条召回）
      </button>
      {open && (
        <div className="max-h-44 space-y-2 overflow-y-auto border-t border-studio-border/40 px-3 py-2">
          {recalls.map((r) => (
            <div key={r.id} className="rounded-lg border border-studio-border/50 bg-studio-bg/40 px-2.5 py-2">
              <div className="flex flex-wrap items-center gap-1.5">
                <span className="text-[11px] font-medium text-studio-text">
                  [{r.category}/{r.subject}]
                </span>
                <span className="rounded bg-studio-panel px-1.5 py-0.5 text-[9px] text-studio-muted">
                  {sourceLabels[r.source] ?? r.source}
                </span>
              </div>
              <p className="mt-1 text-[10px] text-studio-muted">{r.reason}</p>
              <p className="mt-1 line-clamp-3 text-[11px] text-studio-text/85">{r.content}</p>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default function WriteContextPanel({ chapter, volume, defaultCollapsed = true }: Props) {
  const [ctx, setCtx] = useState<WriteContextDTO | null>(null);
  const [loading, setLoading] = useState(false);
  const [collapsed, setCollapsed] = useState(defaultCollapsed);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await app().GetWriteContext(chapter, volume);
      setCtx(data);
    } catch {
      setCtx(null);
    } finally {
      setLoading(false);
    }
  }, [chapter, volume]);

  useEffect(() => {
    setCollapsed(defaultCollapsed);
    load();
  }, [load, defaultCollapsed]);

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
    <div className="flex max-h-64 shrink-0 flex-col overflow-hidden rounded-xl border border-studio-border bg-studio-panel">
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
          <MemoryRecallSection recalls={ctx?.memory_recalls ?? []} />
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
