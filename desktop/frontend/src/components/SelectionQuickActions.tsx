import { useEffect, useRef, useState } from "react";
import { Check, Loader2, Sparkles, Square, Wand2, X } from "lucide-react";
import { SELECTION_EVENTS, eventsOn } from "../lib/runtime";
import { SelectionTransformInput, app } from "../lib/wails";

export type TextSelection = {
  text: string;
  start?: number;
  end?: number;
};

type Props = {
  chapter: number;
  selection: TextSelection | null;
  onApply: (replacement: string) => void;
  onClear: () => void;
};

const actions: { id: SelectionTransformInput["action"]; label: string }[] = [
  { id: "polish", label: "润色" },
  { id: "expand", label: "扩写" },
  { id: "shorten", label: "精简" },
  { id: "dialogue", label: "优化对话" },
];

export default function SelectionQuickActions({ chapter, selection, onApply, onClear }: Props) {
  const [hasKey, setHasKey] = useState(true);
  const [customOpen, setCustomOpen] = useState(false);
  const [customPrompt, setCustomPrompt] = useState("");
  const [running, setRunning] = useState(false);
  const [error, setError] = useState("");
  const [resultOpen, setResultOpen] = useState(false);
  const [result, setResult] = useState("");
  const [activeAction, setActiveAction] = useState("");
  const jobIdRef = useRef("");

  useEffect(() => {
    app()
      .HasAPIKey()
      .then(setHasKey)
      .catch(() => setHasKey(false));
  }, []);

  useEffect(() => {
    setCustomOpen(false);
    setCustomPrompt("");
    setRunning(false);
    setError("");
    setResultOpen(false);
    setResult("");
    setActiveAction("");
    jobIdRef.current = "";
  }, [selection?.text, chapter]);

  useEffect(() => {
    const match = (p: { chapter?: number; job_id?: string }) =>
      p.chapter === chapter && (!jobIdRef.current || p.job_id === jobIdRef.current);

    const unsubs = [
      eventsOn(SELECTION_EVENTS.delta, (p) => {
        if (!match(p)) return;
        setResultOpen(true);
        setResult((prev) => prev + (p.delta || ""));
      }),
      eventsOn(SELECTION_EVENTS.done, (p) => {
        if (!match(p)) return;
        if (p.content) setResult(p.content);
        setRunning(false);
        setResultOpen(true);
      }),
      eventsOn(SELECTION_EVENTS.error, (p) => {
        if (!match(p)) return;
        setError(p.error || "改写失败");
        setRunning(false);
      }),
      eventsOn(SELECTION_EVENTS.status, (p) => {
        if (!match(p)) return;
        if (p.status === "running") setRunning(true);
        if (p.status === "cancelled") setRunning(false);
      }),
    ];
    return () => unsubs.forEach((u) => u());
  }, [chapter]);

  if (!selection?.text.trim()) return null;

  const runTransform = async (action: SelectionTransformInput["action"], prompt = "") => {
    if (!hasKey) {
      setError("请先在设置中配置 API Key");
      return;
    }
    setError("");
    setResult("");
    setActiveAction(action);
    setRunning(true);
    setResultOpen(true);
    setCustomOpen(false);
    try {
      const job = await app().StartSelectionTransform({
        chapter,
        action,
        selected_text: selection.text,
        custom_prompt: prompt,
      });
      jobIdRef.current = job.id;
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setRunning(false);
    }
  };

  const cancel = async () => {
    if (jobIdRef.current) {
      try {
        await app().CancelSelectionTransform(jobIdRef.current);
      } catch {
        /* ignore */
      }
    }
    setRunning(false);
  };

  const apply = () => {
    if (!result.trim()) return;
    onApply(result);
    setResultOpen(false);
    onClear();
  };

  return (
    <>
      <div className="flex shrink-0 flex-wrap items-center gap-2 border-b border-studio-border bg-studio-panel/90 px-3 py-2">
        <span className="text-xs text-studio-muted">
          已选 {selection.text.length} 字
        </span>
        <div className="flex flex-wrap gap-1">
          {actions.map(({ id, label }) => (
            <button
              key={id}
              type="button"
              disabled={running}
              onClick={() => runTransform(id)}
              className="inline-flex items-center gap-1 rounded-md border border-studio-border bg-studio-bg px-2 py-1 text-xs text-studio-text hover:border-studio-accent/40 hover:text-studio-accent disabled:opacity-40"
            >
              <Sparkles className="h-3 w-3" />
              {label}
            </button>
          ))}
          <button
            type="button"
            disabled={running}
            onClick={() => setCustomOpen((v) => !v)}
            className="inline-flex items-center gap-1 rounded-md border border-studio-border bg-studio-bg px-2 py-1 text-xs text-studio-text hover:border-studio-accent/40 hover:text-studio-accent disabled:opacity-40"
          >
            <Wand2 className="h-3 w-3" />
            自定义
          </button>
        </div>
        <button
          type="button"
          onClick={onClear}
          className="ml-auto rounded p-1 text-studio-muted hover:text-studio-text"
          title="取消选择"
        >
          <X className="h-3.5 w-3.5" />
        </button>
      </div>

      {customOpen && (
        <div className="flex shrink-0 items-center gap-2 border-b border-studio-border bg-studio-bg px-3 py-2">
          <input
            value={customPrompt}
            onChange={(e) => setCustomPrompt(e.target.value)}
            placeholder="描述你想如何改写这段文字…"
            className="min-w-0 flex-1 rounded border border-studio-border bg-studio-panel px-2 py-1.5 text-xs outline-none focus:border-studio-accent/50"
            onKeyDown={(e) => e.key === "Enter" && customPrompt.trim() && runTransform("custom", customPrompt.trim())}
          />
          <button
            type="button"
            disabled={!customPrompt.trim() || running}
            onClick={() => runTransform("custom", customPrompt.trim())}
            className="rounded bg-studio-accent px-2 py-1.5 text-xs text-studio-on-accent disabled:opacity-40"
          >
            执行
          </button>
        </div>
      )}

      {error && <div className="shrink-0 px-3 pt-2 studio-alert-error-compact">{error}</div>}

      {resultOpen && (
        <div className="studio-modal-overlay fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="flex max-h-[85vh] w-full max-w-2xl flex-col overflow-hidden rounded-xl border border-studio-border bg-studio-panel shadow-card">
            <div className="flex shrink-0 items-center justify-between border-b border-studio-border px-4 py-3">
              <h3 className="text-sm font-medium">
                {actions.find((a) => a.id === activeAction)?.label || "自定义改写"}
                {running && " · 生成中…"}
              </h3>
              <div className="flex items-center gap-2">
                {running && (
                  <button
                    type="button"
                    onClick={cancel}
                    className="inline-flex items-center gap-1 rounded-lg border border-studio-border px-2 py-1 text-xs text-studio-muted hover:text-studio-text"
                  >
                    <Square className="h-3 w-3" />
                    停止
                  </button>
                )}
                <button type="button" onClick={() => setResultOpen(false)} className="rounded p-1 text-studio-muted hover:text-studio-text">
                  <X className="h-4 w-4" />
                </button>
              </div>
            </div>

            <div className="min-h-0 flex-1 space-y-4 overflow-y-auto p-4">
              <div>
                <p className="mb-1 text-xs font-medium text-studio-muted">原文</p>
                <p className="rounded-lg border border-studio-border bg-studio-bg p-3 text-sm leading-relaxed text-studio-muted">
                  {selection.text}
                </p>
              </div>
              <div>
                <p className="mb-1 text-xs font-medium text-studio-muted">改写结果</p>
                <div className="min-h-[120px] rounded-lg border border-studio-accent/30 bg-studio-bg p-3 text-sm leading-relaxed">
                  {running && !result ? (
                    <span className="inline-flex items-center gap-2 text-studio-muted">
                      <Loader2 className="h-4 w-4 animate-spin" />
                      正在改写…
                    </span>
                  ) : (
                    result || "—"
                  )}
                </div>
              </div>
            </div>

            <div className="flex shrink-0 justify-end gap-2 border-t border-studio-border px-4 py-3">
              <button
                type="button"
                onClick={() => setResultOpen(false)}
                className="rounded-lg px-3 py-1.5 text-xs text-studio-muted hover:text-studio-text"
              >
                关闭
              </button>
              {!running && result.trim() && (
                <>
                  <button
                    type="button"
                    onClick={() => runTransform(activeAction as SelectionTransformInput["action"], customPrompt)}
                    className="rounded-lg border border-studio-border px-3 py-1.5 text-xs hover:bg-studio-bg"
                  >
                    重新生成
                  </button>
                  <button
                    type="button"
                    onClick={apply}
                    className="inline-flex items-center gap-1 rounded-lg bg-studio-accent px-3 py-1.5 text-xs text-studio-on-accent"
                  >
                    <Check className="h-3.5 w-3.5" />
                    替换选中内容
                  </button>
                </>
              )}
            </div>
          </div>
        </div>
      )}
    </>
  );
}
