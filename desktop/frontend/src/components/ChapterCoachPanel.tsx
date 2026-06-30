import { useEffect, useRef, useState } from "react";
import { Check, ChevronDown, ChevronRight, Eye, Loader2, MessageSquare, PanelRightClose, Send, Square, Wand2, X } from "lucide-react";
import { COACH_EVENTS, REVISE_EVENTS, eventsOn } from "../lib/runtime";
import { CoachTurnDTO, DiffResultDTO, app } from "../lib/wails";
import ChapterDiffView from "./ChapterDiffView";

type Props = {
  chapter: number;
  onApplied: () => void;
};

type StreamBubble = {
  thinking: string;
  content: string;
};

export default function ChapterCoachPanel({ chapter, onApplied }: Props) {
  const [turns, setTurns] = useState<CoachTurnDTO[]>([]);
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [hasKey, setHasKey] = useState(true);
  const [streaming, setStreaming] = useState<StreamBubble | null>(null);
  const [thinkingOpen, setThinkingOpen] = useState(true);

  const [reviseJobId, setReviseJobId] = useState("");
  const [reviseRunning, setReviseRunning] = useState(false);
  const [reviseDraft, setReviseDraft] = useState("");
  const [showDraft, setShowDraft] = useState(false);
  const [applying, setApplying] = useState(false);
  const [collapsed, setCollapsed] = useState(true);
  const [showDiffModal, setShowDiffModal] = useState(false);
  const [previewDiff, setPreviewDiff] = useState<DiffResultDTO | null>(null);
  const [previewLoading, setPreviewLoading] = useState(false);

  const scrollRef = useRef<HTMLDivElement>(null);
  const jobIdRef = useRef("");
  const pendingUserRef = useRef<string | null>(null);

  const hasConversation = turns.length > 0;

  useEffect(() => {
    app()
      .HasAPIKey()
      .then(setHasKey)
      .catch(() => setHasKey(false));
  }, []);

  useEffect(() => {
    setTurns([]);
    setInput("");
    setError("");
    setStreaming(null);
    setReviseDraft("");
    setShowDraft(false);
    setReviseJobId("");
    jobIdRef.current = "";
    pendingUserRef.current = null;
    setCollapsed(true);

    app()
      .GetChapterCoachTurns(chapter)
      .then((t) => {
        if (t && t.length > 0) {
          setTurns(t);
        }
      })
      .catch(() => {});
  }, [chapter]);

  useEffect(() => {
    const matchChapter = (p: { chapter?: number }) => p.chapter === chapter;

    const unsubs = [
      eventsOn(COACH_EVENTS.stream, (p) => {
        if (!matchChapter(p)) return;
        const phase = p.phase || "content";
        const delta = p.delta || "";
        setStreaming((prev) => {
          const base = prev ?? { thinking: "", content: "" };
          if (phase === "thinking") {
            return { ...base, thinking: base.thinking + delta };
          }
          return { ...base, content: base.content + delta };
        });
      }),
      eventsOn(COACH_EVENTS.done, (p) => {
        if (!matchChapter(p)) return;
        if (p.turns) {
          try {
            setTurns(JSON.parse(p.turns) as CoachTurnDTO[]);
          } catch {
            /* ignore */
          }
        }
        setStreaming(null);
        setLoading(false);
        pendingUserRef.current = null;
      }),
      eventsOn(COACH_EVENTS.error, (p) => {
        if (!matchChapter(p)) return;
        setError(p.error || "顾问回复失败");
        setStreaming(null);
        setLoading(false);
        if (pendingUserRef.current) {
          setTurns((t) => t.slice(0, -1));
          pendingUserRef.current = null;
        }
      }),
      eventsOn(REVISE_EVENTS.delta, (p) => {
        if (!jobIdRef.current || p.job_id !== jobIdRef.current || p.chapter !== chapter) return;
        setReviseDraft((t) => t + (p.delta || ""));
        setShowDraft(true);
      }),
      eventsOn(REVISE_EVENTS.status, (p) => {
        if (!jobIdRef.current || p.job_id !== jobIdRef.current || p.chapter !== chapter) return;
        if (p.status === "running") setReviseRunning(true);
        if (p.status === "done" || p.status === "failed" || p.status === "cancelled") {
          setReviseRunning(false);
        }
      }),
      eventsOn(REVISE_EVENTS.error, (p) => {
        if (!jobIdRef.current || p.job_id !== jobIdRef.current || p.chapter !== chapter) return;
        setError(p.error || "生成修改稿失败");
        setReviseRunning(false);
      }),
      eventsOn(REVISE_EVENTS.done, (p) => {
        if (!jobIdRef.current || p.job_id !== jobIdRef.current || p.chapter !== chapter) return;
        if (p.content) {
          setReviseDraft(p.content);
          setShowDraft(true);
        }
        setReviseRunning(false);
      }),
    ];
    return () => unsubs.forEach((u) => u());
  }, [chapter]);

  useEffect(() => {
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight });
  }, [turns, streaming, loading]);

  const sendMessage = async () => {
    const msg = input.trim();
    if (!msg || loading) return;
    setError("");
    setInput("");
    setLoading(true);
    setThinkingOpen(true);
    setTurns((t) => [...t, { role: "user", content: msg }]);
    setStreaming({ thinking: "", content: "" });
    pendingUserRef.current = msg;
    try {
      await app().SendChapterCoachMessage(chapter, msg);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setLoading(false);
      setStreaming(null);
      if (pendingUserRef.current) {
        setTurns((t) => t.slice(0, -1));
        pendingUserRef.current = null;
      }
    }
  };

  const startRevision = async () => {
    setError("");
    setReviseDraft("");
    setShowDraft(true);
    try {
      const job = await app().StartChapterRevision(chapter);
      jobIdRef.current = job.id;
      setReviseJobId(job.id);
      setReviseRunning(true);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const cancelRevision = async () => {
    if (!reviseJobId) return;
    try {
      await app().CancelChapterRevision(reviseJobId);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const openApplyPreview = async () => {
    if (!reviseDraft.trim()) return;
    setPreviewLoading(true);
    setError("");
    try {
      const diff = await app().PreviewChapterDiff(chapter, reviseDraft);
      setPreviewDiff(diff);
      setShowDiffModal(true);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setPreviewLoading(false);
    }
  };

  const applyRevision = async () => {
    if (!reviseDraft.trim()) return;
    setApplying(true);
    setError("");
    try {
      await app().ApplyChapterContent(chapter, reviseDraft);
      setShowDiffModal(false);
      setPreviewDiff(null);
      setShowDraft(false);
      setReviseDraft("");
      onApplied();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setApplying(false);
    }
  };

  const resetCoach = async () => {
    await app().ClearChapterCoach(chapter);
    setTurns([]);
    setStreaming(null);
    setReviseDraft("");
    setShowDraft(false);
  };

  if (collapsed) {
    return (
      <button
        type="button"
        onClick={() => setCollapsed(false)}
        className="group flex w-11 shrink-0 flex-col items-center gap-2 self-stretch rounded-xl border border-studio-border bg-studio-panel py-4 text-studio-muted transition hover:border-studio-accent/40 hover:bg-studio-accent/5 hover:text-studio-accent"
        title="展开改稿顾问"
      >
        <MessageSquare className="h-4 w-4 shrink-0" />
        <span className="text-[10px] leading-snug [writing-mode:vertical-rl]">改稿顾问</span>
        {(loading || hasConversation) && (
          <span className="mt-1 h-1.5 w-1.5 shrink-0 rounded-full bg-studio-accent group-hover:brightness-110" />
        )}
      </button>
    );
  }

  return (
    <div className="flex h-full min-h-0 w-72 shrink-0 flex-col overflow-hidden rounded-xl border border-studio-border bg-studio-panel xl:w-80">
      <div className="flex shrink-0 items-center justify-between gap-2 border-b border-studio-border px-4 py-3">
        <div className="flex min-w-0 items-center gap-2 text-sm font-medium">
          <MessageSquare className="h-4 w-4 shrink-0 text-studio-accent" />
          改稿顾问
        </div>
        <div className="flex shrink-0 items-center gap-1">
          {hasConversation && (
            <button
              type="button"
              onClick={resetCoach}
              className="rounded px-2 py-1 text-xs text-studio-muted hover:bg-studio-bg hover:text-studio-text"
            >
              清空
            </button>
          )}
          <button
            type="button"
            onClick={() => setCollapsed(true)}
            className="rounded p-1 text-studio-muted hover:bg-studio-bg hover:text-studio-text"
            title="收起改稿顾问"
          >
            <PanelRightClose className="h-4 w-4" />
          </button>
        </div>
      </div>

      {!hasKey && (
        <div className="mx-4 mt-3 shrink-0 studio-alert-warning-compact">
          未配置 API Key
        </div>
      )}

      {error && (
        <div className="mx-4 mt-3 shrink-0 studio-alert-error-compact">
          {error}
        </div>
      )}

      <div ref={scrollRef} className="min-h-0 flex-1 space-y-3 overflow-y-auto p-4">
        {!hasConversation && !streaming && (
          <p className="py-8 text-center text-sm leading-relaxed text-studio-muted">
            说说本章哪里写得不好、想怎么改
            <br />
            <span className="text-xs">发送第一条消息后，顾问会结合正文与你讨论</span>
          </p>
        )}
        {turns.map((t, i) => (
          <div
            key={i}
            className={`rounded-lg px-3 py-2 text-sm ${
              t.role === "assistant"
                ? "bg-studio-bg text-studio-text"
                : "ml-4 bg-studio-accent/10 text-studio-text"
            }`}
          >
            <p className="mb-1 text-xs text-studio-muted">{t.role === "assistant" ? "顾问" : "你"}</p>
            <div className="whitespace-pre-wrap leading-relaxed">{t.content}</div>
          </div>
        ))}
        {streaming && (
          <div className="rounded-lg bg-studio-bg px-3 py-2 text-sm text-studio-text">
            <p className="mb-1 text-xs text-studio-muted">顾问</p>
            {streaming.thinking && (
              <div className="mb-2 rounded-md border border-studio-border/60 bg-studio-panel/50">
                <button
                  type="button"
                  onClick={() => setThinkingOpen((v) => !v)}
                  className="flex w-full items-center gap-1 px-2 py-1.5 text-xs text-studio-muted hover:text-studio-text"
                >
                  {thinkingOpen ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
                  思考过程
                  {loading && !streaming.content && <Loader2 className="ml-1 h-3 w-3 animate-spin" />}
                </button>
                {thinkingOpen && (
                  <div className="whitespace-pre-wrap border-t border-studio-border/40 px-2 py-2 text-xs leading-relaxed text-studio-muted">
                    {streaming.thinking}
                  </div>
                )}
              </div>
            )}
            <div className="whitespace-pre-wrap leading-relaxed">
              {streaming.content}
              {loading && !streaming.content && !streaming.thinking && (
                <span className="inline-flex items-center gap-1 text-xs text-studio-muted">
                  <Loader2 className="h-3 w-3 animate-spin" />
                  思考中…
                </span>
              )}
            </div>
          </div>
        )}
      </div>

      {showDraft && (
        <div className="mx-4 mb-3 shrink-0 rounded-lg border border-studio-accent/30 bg-studio-paper">
          <div className="flex items-center justify-between border-b border-studio-border px-3 py-2">
            <span className="text-xs font-medium text-studio-accent">修改稿预览</span>
            <button type="button" onClick={() => setShowDraft(false)} className="text-studio-muted hover:text-studio-text">
              <X className="h-4 w-4" />
            </button>
          </div>
          <pre className="max-h-40 overflow-y-auto whitespace-pre-wrap p-3 font-serif text-xs leading-relaxed text-studio-ink">
            {reviseDraft || (reviseRunning ? "生成中…" : "")}
          </pre>
          {reviseDraft && !reviseRunning && (
            <div className="flex gap-2 border-t border-studio-border p-2">
              <button
                type="button"
                onClick={openApplyPreview}
                disabled={previewLoading || applying}
                className="inline-flex flex-1 items-center justify-center gap-1 rounded-lg border border-studio-accent/40 px-3 py-1.5 text-xs text-studio-accent hover:bg-studio-accent/10 disabled:opacity-40"
              >
                {previewLoading ? <Loader2 className="h-3 w-3 animate-spin" /> : <Eye className="h-3 w-3" />}
                预览变更
              </button>
            </div>
          )}
        </div>
      )}

      <div className="shrink-0 space-y-2 border-t border-studio-border p-3">
        <div className="flex gap-2">
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                sendMessage();
              }
            }}
            disabled={loading || !hasKey}
            placeholder="说说哪里不满意…"
            rows={2}
            className="min-h-0 flex-1 resize-none rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none focus:border-studio-accent disabled:opacity-50"
          />
          <button
            type="button"
            onClick={sendMessage}
            disabled={loading || !input.trim() || !hasKey}
            className="self-end rounded-lg bg-studio-accent px-3 py-2 text-studio-on-accent hover:brightness-110 disabled:opacity-40"
          >
            <Send className="h-4 w-4" />
          </button>
        </div>
        <div className="flex gap-2">
          {reviseRunning ? (
            <button
              type="button"
              onClick={cancelRevision}
              className="inline-flex flex-1 items-center justify-center gap-1 rounded-lg border border-[rgb(var(--studio-danger-border))] px-3 py-1.5 text-xs text-[rgb(var(--studio-danger-fg))] hover:bg-[rgb(var(--studio-danger-bg))]"
            >
              <Square className="h-3 w-3" />
              取消生成
            </button>
          ) : (
            <button
              type="button"
              onClick={startRevision}
              disabled={!hasKey || turns.length < 2 || loading}
              className="inline-flex flex-1 items-center justify-center gap-1 rounded-lg border border-studio-accent/40 px-3 py-1.5 text-xs text-studio-accent hover:bg-studio-accent/10 disabled:opacity-40"
            >
              <Wand2 className="h-3 w-3" />
              生成修改稿
            </button>
          )}
        </div>
      </div>

      {showDiffModal && previewDiff && (
        <div className="studio-modal-overlay fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="flex max-h-[85vh] w-full max-w-xl flex-col overflow-hidden rounded-xl border border-studio-border bg-studio-panel shadow-card">
            <div className="flex shrink-0 items-center justify-between border-b border-studio-border px-4 py-3">
              <span className="text-sm font-medium">应用修改稿 · 变更预览</span>
              <button
                type="button"
                onClick={() => setShowDiffModal(false)}
                className="rounded p-1 text-studio-muted hover:bg-studio-bg hover:text-studio-text"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
            <div className="min-h-0 flex-1 overflow-y-auto p-4">
              <ChapterDiffView diff={previewDiff} maxHeight="max-h-[55vh]" />
            </div>
            <div className="flex shrink-0 gap-2 border-t border-studio-border p-3">
              <button
                type="button"
                onClick={() => setShowDiffModal(false)}
                className="flex-1 rounded-lg border border-studio-border px-3 py-2 text-sm text-studio-muted hover:bg-studio-bg"
              >
                取消
              </button>
              <button
                type="button"
                onClick={applyRevision}
                disabled={applying}
                className="inline-flex flex-1 items-center justify-center gap-1 rounded-lg bg-studio-accent px-3 py-2 text-sm font-medium text-studio-on-accent hover:brightness-110 disabled:opacity-40"
              >
                {applying ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />}
                确认应用
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
