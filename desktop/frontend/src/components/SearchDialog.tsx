import { useEffect, useRef, useState } from "react";
import { Loader2, Search, X } from "lucide-react";
import { SearchHitDTO, app } from "../lib/wails";

export type SearchSession = {
  query: string;
  results: SearchHitDTO[];
};

type Props = {
  open: boolean;
  onClose: () => void;
  session: SearchSession | null;
  onSessionChange: (session: SearchSession | null) => void;
  onNavigate: (hit: SearchHitDTO) => void;
};

const kindLabel: Record<string, string> = {
  chapter: "章节",
  setting: "设定",
  memory: "记忆",
  foreshadow: "伏笔",
  entity: "实体",
};

export default function SearchDialog({
  open,
  onClose,
  session,
  onSessionChange,
  onNavigate,
}: Props) {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<SearchHitDTO[]>([]);
  const [loading, setLoading] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!open) return;
    setQuery(session?.query ?? "");
    setResults(session?.results ?? []);
    setTimeout(() => inputRef.current?.focus(), 50);
  }, [open, session]);

  useEffect(() => {
    if (!open) return;
    const q = query.trim();
    if (!q) {
      setResults([]);
      onSessionChange(null);
      return;
    }
    const timer = setTimeout(async () => {
      setLoading(true);
      try {
        const hits = await app().SearchProject(q, 25);
        const next = hits ?? [];
        setResults(next);
        onSessionChange({ query: q, results: next });
      } catch {
        setResults([]);
        onSessionChange({ query: q, results: [] });
      } finally {
        setLoading(false);
      }
    }, 250);
    return () => clearTimeout(timer);
  }, [query, open, onSessionChange]);

  const openHit = (hit: SearchHitDTO) => {
    onNavigate(hit);
    onClose();
  };

  if (!open) return null;

  return (
    <div
      className="studio-modal-overlay fixed inset-0 z-50 flex items-start justify-center p-4 pt-[12vh]"
      onClick={onClose}
    >
      <div
        className="w-full max-w-xl overflow-hidden rounded-xl border border-studio-border bg-studio-panel shadow-card"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center gap-2 border-b border-studio-border px-4 py-3">
          <Search className="h-4 w-4 shrink-0 text-studio-muted" />
          <input
            ref={inputRef}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="搜索章节、设定、记忆、伏笔…"
            className="min-w-0 flex-1 bg-transparent text-sm outline-none placeholder:text-studio-muted/70"
            onKeyDown={(e) => e.key === "Escape" && onClose()}
          />
          <button type="button" onClick={onClose} className="rounded p-1 text-studio-muted hover:text-studio-text">
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="max-h-[50vh] overflow-y-auto">
          {loading ? (
            <div className="flex items-center justify-center py-10 text-studio-muted">
              <Loader2 className="h-5 w-5 animate-spin" />
            </div>
          ) : !query.trim() ? (
            <p className="py-10 text-center text-sm text-studio-muted">输入关键词开始搜索</p>
          ) : results.length === 0 ? (
            <p className="py-10 text-center text-sm text-studio-muted">无匹配结果</p>
          ) : (
            <ul>
              {results.map((hit) => (
                <li key={`${hit.kind}-${hit.id}`}>
                  <button
                    type="button"
                    onClick={() => openHit(hit)}
                    className="w-full border-b border-studio-border px-4 py-3 text-left transition hover:bg-studio-bg"
                  >
                    <div className="flex items-center gap-2">
                      <span className="rounded bg-studio-border px-1.5 py-0.5 text-[10px] text-studio-muted">
                        {kindLabel[hit.kind] || hit.kind}
                      </span>
                      <span className="truncate text-sm font-medium">{hit.title || hit.id}</span>
                      {hit.chapter > 0 && (
                        <span className="ml-auto shrink-0 text-xs text-studio-muted">第{hit.chapter}章</span>
                      )}
                    </div>
                    {hit.snippet && (
                      <p className="mt-1 line-clamp-2 text-xs text-studio-muted">{hit.snippet}</p>
                    )}
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  );
}
