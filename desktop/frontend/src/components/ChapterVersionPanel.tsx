import { useCallback, useEffect, useState } from "react";
import { History, Loader2, RotateCcw, X } from "lucide-react";
import { DiffResultDTO, VersionEntryDTO, app, formatRelativeTime } from "../lib/wails";
import ChapterDiffView from "./ChapterDiffView";

type Props = {
  chapter: number;
  open: boolean;
  onClose: () => void;
  onRestored: () => void;
};

const sourceLabel: Record<string, string> = {
  write_draft: "写章起草",
  write_review: "审查润色",
  coach_apply: "改稿应用",
  restore: "版本恢复",
  coach_draft: "改稿草稿",
  manual: "手动备份",
};

export default function ChapterVersionPanel({ chapter, open, onClose, onRestored }: Props) {
  const [versions, setVersions] = useState<VersionEntryDTO[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedID, setSelectedID] = useState("");
  const [diff, setDiff] = useState<DiffResultDTO | null>(null);
  const [diffLoading, setDiffLoading] = useState(false);
  const [restoring, setRestoring] = useState(false);
  const [error, setError] = useState("");

  const loadVersions = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const list = await app().ListChapterVersions(chapter);
      setVersions(list);
      if (list.length > 0) {
        setSelectedID((prev) => prev || list[0].id);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [chapter]);

  useEffect(() => {
    if (!open) return;
    setSelectedID("");
    setDiff(null);
    loadVersions();
  }, [open, chapter, loadVersions]);

  useEffect(() => {
    if (!open || !selectedID) {
      setDiff(null);
      return;
    }
    let cancelled = false;
    setDiffLoading(true);
    app()
      .DiffChapterVersions(chapter, selectedID, "current")
      .then((d) => {
        if (!cancelled) setDiff(d);
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e));
      })
      .finally(() => {
        if (!cancelled) setDiffLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [open, chapter, selectedID]);

  const restore = async () => {
    if (!selectedID) return;
    if (!confirm(`确定将正文恢复为 ${selectedID}？当前正文会先自动备份。`)) return;
    setRestoring(true);
    setError("");
    try {
      await app().RestoreChapterVersion(chapter, selectedID);
      onRestored();
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setRestoring(false);
    }
  };

  if (!open) return null;

  return (
    <div className="studio-modal-overlay fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="flex max-h-[85vh] w-full max-w-2xl flex-col overflow-hidden rounded-xl border border-studio-border bg-studio-panel shadow-card">
        <div className="flex shrink-0 items-center justify-between border-b border-studio-border px-4 py-3">
          <div className="flex items-center gap-2 text-sm font-medium">
            <History className="h-4 w-4 text-studio-accent" />
            第{chapter}章 · 版本历史
          </div>
          <button type="button" onClick={onClose} className="rounded p-1 text-studio-muted hover:bg-studio-bg hover:text-studio-text">
            <X className="h-4 w-4" />
          </button>
        </div>

        {error && (
          <div className="mx-4 mt-3 shrink-0 studio-alert-error-compact">
            {error}
          </div>
        )}

        <div className="flex min-h-0 flex-1 overflow-hidden">
          <div className="w-52 shrink-0 overflow-y-auto border-r border-studio-border">
            {loading ? (
              <div className="flex items-center justify-center p-8 text-studio-muted">
                <Loader2 className="h-5 w-5 animate-spin" />
              </div>
            ) : versions.length === 0 ? (
              <p className="p-4 text-sm text-studio-muted">暂无历史版本</p>
            ) : (
              <ul>
                {versions.map((v) => (
                  <li key={v.id}>
                    <button
                      type="button"
                      onClick={() => setSelectedID(v.id)}
                      className={`w-full border-b border-studio-border px-3 py-2.5 text-left text-sm transition hover:bg-studio-bg ${
                        selectedID === v.id ? "bg-studio-bg text-studio-accent" : ""
                      }`}
                    >
                      <div className="font-medium">{v.id}</div>
                      <div className="mt-0.5 text-xs text-studio-muted">{v.label || sourceLabel[v.source] || v.source}</div>
                      <div className="mt-0.5 text-[10px] text-studio-muted/80">
                        {formatRelativeTime(v.created_at)} · {v.word_count} 字
                      </div>
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>

          <div className="min-w-0 flex-1 overflow-y-auto p-4">
            {diffLoading ? (
              <div className="flex items-center justify-center py-12 text-studio-muted">
                <Loader2 className="h-5 w-5 animate-spin" />
              </div>
            ) : diff ? (
              <>
                <ChapterDiffView diff={diff} maxHeight="max-h-[50vh]" />
                {selectedID && (
                  <div className="mt-4 flex justify-end">
                    <button
                      type="button"
                      onClick={restore}
                      disabled={restoring}
                      className="inline-flex items-center gap-1.5 rounded-lg border border-studio-accent/40 px-4 py-2 text-sm text-studio-accent hover:bg-studio-accent/10 disabled:opacity-40"
                    >
                      {restoring ? <Loader2 className="h-4 w-4 animate-spin" /> : <RotateCcw className="h-4 w-4" />}
                      恢复此版本
                    </button>
                  </div>
                )}
              </>
            ) : selectedID ? null : (
              <p className="py-8 text-center text-sm text-studio-muted">选择左侧版本查看与当前正文的差异</p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
