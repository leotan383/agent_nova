import { useCallback, useEffect, useRef, useState } from "react";
import { Eye, Loader2, Pencil, Save } from "lucide-react";
import ContentPreview from "./ContentPreview";
import SelectionQuickActions, { TextSelection } from "./SelectionQuickActions";

type ViewMode = "preview" | "edit";

type Props = {
  value: string;
  editable?: boolean;
  paper?: boolean;
  saving?: boolean;
  onSave?: (content: string) => Promise<void>;
  emptyHint?: string;
  /** 传入章号时，正文选区可触发 AI 快捷改写 */
  selectionChapter?: number;
};

export default function MarkdownEditor({
  value,
  editable = true,
  paper = false,
  saving = false,
  onSave,
  emptyHint = "暂无内容",
  selectionChapter,
}: Props) {
  const [mode, setMode] = useState<ViewMode>("preview");
  const [draft, setDraft] = useState(value);
  const [error, setError] = useState("");
  const [selection, setSelection] = useState<TextSelection | null>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const previewRef = useRef<HTMLDivElement>(null);

  const selectionEnabled = !!selectionChapter && editable && !!onSave;

  useEffect(() => {
    setDraft(value);
    setMode("preview");
    setError("");
    setSelection(null);
  }, [value]);

  const dirty = draft !== value;
  const canEdit = editable && !!onSave;

  const handleSave = async () => {
    if (!onSave || !dirty) return;
    setError("");
    try {
      await onSave(draft);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const readTextareaSelection = useCallback(() => {
    const el = textareaRef.current;
    if (!el) return;
    const start = el.selectionStart;
    const end = el.selectionEnd;
    if (start === end) {
      setSelection(null);
      return;
    }
    setSelection({
      text: el.value.slice(start, end),
      start,
      end,
    });
  }, []);

  const readPreviewSelection = useCallback(() => {
    const sel = window.getSelection();
    if (!sel || sel.isCollapsed || !previewRef.current) {
      setSelection(null);
      return;
    }
    const text = sel.toString().trim();
    if (!text || !previewRef.current.contains(sel.anchorNode)) {
      setSelection(null);
      return;
    }
    setSelection({ text: sel.toString() });
  }, []);

  const applySelectionReplace = (replacement: string) => {
    if (selection?.start != null && selection.end != null) {
      const next = draft.slice(0, selection.start) + replacement + draft.slice(selection.end);
      setDraft(next);
      setSelection(null);
      return;
    }
    const idx = draft.indexOf(selection?.text ?? "");
    if (idx < 0) {
      setError("无法在正文中定位选中片段，请切换到编辑模式后重试");
      return;
    }
    const next = draft.slice(0, idx) + replacement + draft.slice(idx + (selection?.text.length ?? 0));
    setDraft(next);
    setSelection(null);
  };

  const clearSelection = () => {
    setSelection(null);
    window.getSelection()?.removeAllRanges();
  };

  return (
    <div className="flex h-full min-h-0 flex-1 flex-col overflow-hidden">
      <div className="flex shrink-0 items-center justify-between gap-2 border-b border-studio-border px-3 py-2">
        <div className="flex gap-1 rounded-lg border border-studio-border bg-studio-bg p-0.5">
          <button
            type="button"
            onClick={() => {
              setSelection(null);
              setMode("preview");
            }}
            className={`inline-flex items-center gap-1 rounded-md px-2.5 py-1 text-xs transition ${
              mode === "preview"
                ? "bg-studio-panel text-studio-text shadow-sm"
                : "text-studio-muted hover:text-studio-text"
            }`}
          >
            <Eye className="h-3.5 w-3.5" />
            预览
          </button>
          {canEdit && (
            <button
              type="button"
              onClick={() => {
                setSelection(null);
                setMode("edit");
              }}
              className={`inline-flex items-center gap-1 rounded-md px-2.5 py-1 text-xs transition ${
                mode === "edit"
                  ? "bg-studio-panel text-studio-text shadow-sm"
                  : "text-studio-muted hover:text-studio-text"
              }`}
            >
              <Pencil className="h-3.5 w-3.5" />
              编辑
            </button>
          )}
        </div>

        {canEdit && (
          <button
            type="button"
            onClick={handleSave}
            disabled={!dirty || saving}
            className="inline-flex items-center gap-1 rounded-lg bg-studio-accent px-3 py-1.5 text-xs font-medium text-studio-on-accent hover:brightness-110 disabled:opacity-40"
          >
            {saving ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Save className="h-3.5 w-3.5" />}
            保存
          </button>
        )}
      </div>

      {selectionEnabled && selection && (
        <SelectionQuickActions
          chapter={selectionChapter}
          selection={selection}
          onApply={applySelectionReplace}
          onClear={clearSelection}
        />
      )}

      {error && <div className="shrink-0 px-3 pt-2 studio-alert-error-compact">{error}</div>}

      {mode === "edit" && canEdit ? (
        <textarea
          ref={textareaRef}
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onSelect={readTextareaSelection}
          onMouseUp={readTextareaSelection}
          onKeyUp={readTextareaSelection}
          spellCheck={false}
          className={`min-h-0 flex-1 resize-none border-0 bg-studio-bg p-4 font-mono text-sm leading-relaxed text-studio-text outline-none ${
            paper ? "bg-studio-paper text-studio-ink" : ""
          }`}
          placeholder="在此编辑 Markdown…"
        />
      ) : (
        <div
          ref={previewRef}
          onMouseUp={selectionEnabled ? readPreviewSelection : undefined}
          className={`min-h-0 flex-1 overflow-y-auto p-6 ${paper ? "bg-studio-paper text-studio-ink" : ""}`}
        >
          {draft.trim() ? (
            <ContentPreview content={draft} paper={paper} />
          ) : (
            <p className="text-sm text-studio-muted/70">{emptyHint}</p>
          )}
        </div>
      )}
    </div>
  );
}
