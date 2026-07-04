import { useEffect, useState } from "react";
import { AlertTriangle, Loader2 } from "lucide-react";
import {
  ChapterStructurePreviewDTO,
  InsertChapterInput,
  app,
} from "../lib/wails";

type Mode = "insert" | "delete";

type Props = {
  open: boolean;
  mode: Mode;
  chapter: number;
  onClose: () => void;
  onDone: () => void;
};

export default function ChapterStructureDialog({
  open,
  mode,
  chapter,
  onClose,
  onDone,
}: Props) {
  const [title, setTitle] = useState("");
  const [preview, setPreview] = useState<ChapterStructurePreviewDTO | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!open) return;
    setTitle("");
    setError("");
    setLoading(true);
    const load = async () => {
      try {
        if (mode === "insert") {
          const p = await app().PreviewInsertChapter({
            after_chapter: chapter,
            title: "新章",
          } satisfies InsertChapterInput);
          setPreview(p);
        } else {
          setPreview(await app().PreviewDeleteChapter(chapter));
        }
      } catch (e) {
        setError(e instanceof Error ? e.message : String(e));
        setPreview(null);
      } finally {
        setLoading(false);
      }
    };
    void load();
  }, [open, mode, chapter]);

  useEffect(() => {
    if (!open || mode !== "insert") return;
    const t = setTimeout(() => {
      void app()
        .PreviewInsertChapter({ after_chapter: chapter, title: title || "新章" })
        .then(setPreview)
        .catch(() => {});
    }, 300);
    return () => clearTimeout(t);
  }, [open, mode, chapter, title]);

  if (!open) return null;

  const submit = async () => {
    setSaving(true);
    setError("");
    try {
      if (mode === "insert") {
        await app().InsertChapterAfter({ after_chapter: chapter, title: title || "新章" });
      } else {
        await app().DeleteChapter({ chapter });
      }
      onDone();
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="studio-modal-overlay fixed inset-0 z-[70] flex items-center justify-center p-4" onClick={onClose}>
      <div
        className="w-full max-w-md rounded-xl border border-studio-border bg-studio-panel shadow-card"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="border-b border-studio-border px-5 py-4">
          <h2 className="text-base font-medium">
            {mode === "insert" ? `在第 ${chapter} 章后插入` : `删除第 ${chapter} 章`}
          </h2>
        </div>
        <div className="space-y-4 px-5 py-4 text-sm">
          {mode === "insert" && (
            <label className="block">
              <span className="text-xs text-studio-muted">新章标题</span>
              <input
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                className="mt-1 w-full rounded-lg border border-studio-border bg-studio-bg px-3 py-2 outline-none"
                placeholder="如：插曲"
              />
            </label>
          )}
          {loading ? (
            <div className="flex items-center gap-2 text-studio-muted">
              <Loader2 className="h-4 w-4 animate-spin" />
              计算影响…
            </div>
          ) : preview ? (
            <div className="space-y-2 rounded-lg border border-studio-border bg-studio-bg/50 px-3 py-2 text-xs">
              <p className="font-medium text-studio-text">将影响：</p>
              <ul className="space-y-1 text-studio-muted">
                {preview.impact.chapters_shifted > 0 && (
                  <li>· {preview.impact.chapters_shifted} 个章节目录重编号</li>
                )}
                {preview.impact.memories_affected > 0 && (
                  <li>· {preview.impact.memories_affected} 条记忆章号更新</li>
                )}
                {preview.impact.foreshadows_affected > 0 && (
                  <li>· {preview.impact.foreshadows_affected} 条伏笔章号更新</li>
                )}
                {preview.open_foreshadows_at_target > 0 && mode === "delete" && (
                  <li className="text-amber-600">
                    · {preview.open_foreshadows_at_target} 条开放伏笔埋在此章
                  </li>
                )}
              </ul>
            </div>
          ) : null}
          {mode === "delete" && (
            <div className="flex items-start gap-2 rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-700 dark:text-amber-300">
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
              删章不可撤销，建议先在 Overview 备份项目。
            </div>
          )}
          {error && <p className="text-xs text-[rgb(var(--studio-danger-fg))]">{error}</p>}
        </div>
        <div className="flex gap-2 border-t border-studio-border px-5 py-3">
          <button
            type="button"
            onClick={onClose}
            className="flex-1 rounded-lg border border-studio-border px-3 py-2 text-sm text-studio-muted hover:bg-studio-bg"
          >
            取消
          </button>
          <button
            type="button"
            disabled={saving || loading}
            onClick={() => void submit()}
            className="flex-1 rounded-lg bg-studio-accent px-3 py-2 text-sm font-medium text-studio-on-accent disabled:opacity-40"
          >
            {saving ? "处理中…" : "确认"}
          </button>
        </div>
      </div>
    </div>
  );
}
