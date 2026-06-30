import { useCallback, useEffect, useState } from "react";
import { ChevronDown, ChevronRight, Loader2, RefreshCw } from "lucide-react";
import { WriteContextDTO, app } from "../lib/wails";
import ContentPreview from "./ContentPreview";

type Props = {
  chapter: number;
  volume: number;
};

function Section({ title, body, defaultOpen = false }: { title: string; body: string; defaultOpen?: boolean }) {
  const [open, setOpen] = useState(defaultOpen);
  if (!body.trim()) return null;
  return (
    <div className="border-b border-studio-border/60">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center gap-2 px-3 py-2 text-left text-xs font-medium text-studio-muted hover:bg-studio-bg hover:text-studio-text"
      >
        {open ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
        {title}
      </button>
      {open && (
        <div className="max-h-48 overflow-y-auto border-t border-studio-border/40 px-3 py-2">
          <ContentPreview content={body} className="text-xs" />
        </div>
      )}
    </div>
  );
}

export default function WriteContextPanel({ chapter, volume }: Props) {
  const [ctx, setCtx] = useState<WriteContextDTO | null>(null);
  const [loading, setLoading] = useState(false);
  const [collapsed, setCollapsed] = useState(false);

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
    load();
  }, [load]);

  if (collapsed) {
    return (
      <button
        type="button"
        onClick={() => setCollapsed(false)}
        className="shrink-0 rounded-xl border border-studio-border bg-studio-panel px-3 py-2 text-xs text-studio-muted hover:text-studio-text"
      >
        展开本章上下文
      </button>
    );
  }

  return (
    <div className="flex max-h-72 shrink-0 flex-col overflow-hidden rounded-xl border border-studio-border bg-studio-panel">
      <div className="flex shrink-0 items-center justify-between border-b border-studio-border px-3 py-2">
        <span className="text-xs font-medium text-studio-muted">
          第{chapter}章写作上下文
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
          <Section title="近章摘要" body={ctx?.recent_summary ?? ""} defaultOpen />
          <Section title="设定摘要" body={ctx?.settings ?? ""} />
          <Section title="长期记忆" body={ctx?.memories ?? ""} />
          <Section title="Open 伏笔" body={ctx?.open_foreshadows ?? ""} />
          <Section title="检索命中" body={ctx?.fts_hits ?? ""} />
        </div>
      )}
    </div>
  );
}
