import { useCallback, useEffect, useState } from "react";
import { Check, Loader2, Pencil } from "lucide-react";
import { ForeshadowDTO, app } from "../lib/wails";

export type ForeshadowFocus = "open" | "resolved";

type Props = {
  focus: ForeshadowFocus;
  onFocusChange?: (focus: ForeshadowFocus) => void;
  highlightId?: string;
};

const focusTabs: { id: ForeshadowFocus; label: string }[] = [
  { id: "open", label: "Open 伏笔" },
  { id: "resolved", label: "已回收" },
];

export default function ForeshadowPanel({ focus, onFocusChange, highlightId = "" }: Props) {
  const [foreshadows, setForeshadows] = useState<ForeshadowDTO[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [editingForeshadow, setEditingForeshadow] = useState<ForeshadowDTO | null>(null);
  const [saving, setSaving] = useState(false);
  const [resolveID, setResolveID] = useState("");
  const [resolveChapter, setResolveChapter] = useState(0);

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const status = focus === "resolved" ? "resolved" : "open";
      const list = await app().ListForeshadows(status);
      setForeshadows(list ?? []);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [focus]);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    if (!highlightId) return;
    const timer = setTimeout(() => {
      document.getElementById(`foreshadow-item-${highlightId}`)?.scrollIntoView({ block: "center", behavior: "smooth" });
    }, 100);
    return () => clearTimeout(timer);
  }, [highlightId, foreshadows, focus]);

  const saveForeshadow = async () => {
    if (!editingForeshadow) return;
    if (!editingForeshadow.description.trim()) {
      setError("伏笔描述不能为空");
      return;
    }
    setSaving(true);
    try {
      await app().UpdateForeshadow(editingForeshadow.id, editingForeshadow.description.trim());
      setEditingForeshadow(null);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  };

  const resolveForeshadow = async (id: string) => {
    try {
      await app().ResolveForeshadow(id, resolveChapter);
      setResolveID("");
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const tabBar = (
    <div className="mb-4 flex flex-wrap gap-1 rounded-lg border border-studio-border bg-studio-bg p-1">
      {focusTabs.map(({ id, label }) => (
        <button
          key={id}
          type="button"
          onClick={() => onFocusChange?.(id)}
          disabled={!onFocusChange}
          className={`rounded-md px-2.5 py-1.5 text-xs transition ${
            focus === id
              ? "bg-studio-panel text-studio-accent shadow-sm"
              : "text-studio-muted hover:text-studio-text"
          }`}
        >
          {label}
        </button>
      ))}
    </div>
  );

  if (loading) {
    return (
      <div className="flex items-center gap-2 text-sm text-studio-muted">
        <Loader2 className="h-4 w-4 animate-spin" />
        加载中…
      </div>
    );
  }

  const isOpen = focus === "open";
  const title = isOpen ? "Open 伏笔" : "已回收伏笔";

  return (
    <div className="min-h-0 flex-1 overflow-y-auto">
      {tabBar}
      <h2 className="mb-1 text-sm font-medium text-studio-muted">
        {title} ({foreshadows.length})
      </h2>
      <p className="mb-4 text-xs leading-relaxed text-studio-muted">
        伏笔是连载中的悬念债务：Open 表示尚未在正文兑现；已回收表示已在某章完成收束。
      </p>
      {error && <div className="mb-4 studio-alert-error-compact">{error}</div>}
      {foreshadows.length === 0 ? (
        <p className="text-sm text-studio-muted">{isOpen ? "暂无 open 伏笔" : "暂无已回收伏笔"}</p>
      ) : (
        <ul className="space-y-3">
          {foreshadows.map((f) => (
            <li
              key={f.id}
              id={`foreshadow-item-${f.id}`}
              className={`rounded-xl border bg-studio-panel p-4 ${
                highlightId === f.id
                  ? "border-studio-accent ring-2 ring-studio-accent/30"
                  : "border-studio-border"
              }`}
            >
              {editingForeshadow?.id === f.id ? (
                <div className="space-y-2">
                  <textarea
                    value={editingForeshadow.description}
                    onChange={(e) => setEditingForeshadow({ ...editingForeshadow, description: e.target.value })}
                    rows={3}
                    className="w-full resize-none rounded border border-studio-border bg-studio-bg px-2 py-1.5 text-sm outline-none"
                  />
                  <div className="flex gap-2">
                    <button
                      type="button"
                      onClick={saveForeshadow}
                      disabled={saving}
                      className="rounded-lg bg-studio-accent px-3 py-1 text-xs text-studio-on-accent disabled:opacity-40"
                    >
                      {saving ? <Loader2 className="h-3 w-3 animate-spin" /> : "保存"}
                    </button>
                    <button type="button" onClick={() => setEditingForeshadow(null)} className="text-xs text-studio-muted">
                      取消
                    </button>
                  </div>
                </div>
              ) : (
                <>
                  <p className="text-sm leading-relaxed">{f.description}</p>
                  <p className="mt-2 text-xs text-studio-muted">
                    埋设于第 {f.planted_chapter} 章
                    {!isOpen && f.resolved_chapter > 0 && ` · 回收于第 ${f.resolved_chapter} 章`}
                    {" · "}
                    {f.id}
                  </p>
                  <div className="mt-3 flex flex-wrap gap-3">
                    {isOpen && (
                      <>
                        {resolveID === f.id ? (
                          <div className="flex flex-wrap items-center gap-2">
                            <input
                              type="number"
                              min={1}
                              value={resolveChapter || ""}
                              onChange={(e) => setResolveChapter(Number(e.target.value))}
                              placeholder="回收章号"
                              className="w-24 rounded border border-studio-border bg-studio-bg px-2 py-1 text-xs outline-none"
                            />
                            <button
                              type="button"
                              onClick={() => resolveForeshadow(f.id)}
                              className="inline-flex items-center gap-1 rounded bg-studio-accent px-2 py-1 text-xs text-studio-on-accent"
                            >
                              <Check className="h-3 w-3" />
                              确认
                            </button>
                            <button type="button" onClick={() => setResolveID("")} className="text-xs text-studio-muted">
                              取消
                            </button>
                          </div>
                        ) : (
                          <button
                            type="button"
                            onClick={() => {
                              setResolveID(f.id);
                              setResolveChapter(f.planted_chapter);
                            }}
                            className="inline-flex items-center gap-1 text-xs text-studio-accent hover:underline"
                          >
                            <Check className="h-3 w-3" />
                            标记已回收
                          </button>
                        )}
                      </>
                    )}
                    <button
                      type="button"
                      onClick={() => setEditingForeshadow({ ...f })}
                      className="inline-flex items-center gap-1 text-xs text-studio-muted hover:text-studio-text"
                    >
                      <Pencil className="h-3 w-3" />
                      编辑描述
                    </button>
                  </div>
                </>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
