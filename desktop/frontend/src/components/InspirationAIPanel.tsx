import { useEffect, useRef, useState } from "react";
import { Check, Loader2, Send, Sparkles, Wand2, X } from "lucide-react";
import { INSPIRATION_EVENTS, eventsOn } from "../lib/runtime";
import {
  CoachTurnDTO,
  InspirationDTO,
  InspirationEnrichPreviewDTO,
  app,
} from "../lib/wails";
import { tagsToInput } from "../lib/inspirationUtils";

type Props = {
  inspirationId: string;
  onApplied: (updated: InspirationDTO) => void;
};

export default function InspirationAIPanel({ inspirationId, onApplied }: Props) {
  const [turns, setTurns] = useState<CoachTurnDTO[]>([]);
  const [input, setInput] = useState("");
  const [discussing, setDiscussing] = useState(false);
  const [loading, setLoading] = useState(false);
  const [enriching, setEnriching] = useState(false);
  const [streaming, setStreaming] = useState("");
  const [error, setError] = useState("");
  const [preview, setPreview] = useState<InspirationEnrichPreviewDTO | null>(null);
  const [applying, setApplying] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight });
  }, [turns, streaming, preview]);

  useEffect(() => {
    const unsubs = [
      eventsOn(INSPIRATION_EVENTS.stream, (p) => {
        setStreaming((t) => t + (p.delta || ""));
      }),
      eventsOn(INSPIRATION_EVENTS.done, (p) => {
        if (p.turns) {
          try {
            setTurns(JSON.parse(p.turns) as CoachTurnDTO[]);
          } catch {
            /* ignore */
          }
        }
        setStreaming("");
        setLoading(false);
      }),
      eventsOn(INSPIRATION_EVENTS.error, (p) => {
        setError(p.error || "AI 处理失败");
        setStreaming("");
        setLoading(false);
        setEnriching(false);
      }),
    ];
    return () => {
      unsubs.forEach((u) => u());
      app().ClearInspirationDiscuss().catch(() => {});
    };
  }, []);

  const startDiscuss = async () => {
    setError("");
    setPreview(null);
    setLoading(true);
    setStreaming("");
    try {
      const t = await app().StartInspirationDiscuss(inspirationId);
      setTurns(t ?? []);
      setDiscussing(true);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  };

  const send = async () => {
    const msg = input.trim();
    if (!msg || loading) return;
    setInput("");
    setError("");
    setTurns((prev) => [...prev, { role: "user", content: msg }]);
    setLoading(true);
    setStreaming("");
    try {
      await app().SendInspirationDiscussMessage(msg);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setLoading(false);
    }
  };

  const finishDiscuss = async () => {
    setError("");
    setLoading(true);
    try {
      const p = await app().FinishInspirationDiscuss();
      setPreview(p);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  };

  const runEnrich = async () => {
    setError("");
    setPreview(null);
    setEnriching(true);
    try {
      const p = await app().EnrichInspirationWithAI(inspirationId);
      setPreview(p);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setEnriching(false);
    }
  };

  const applyPreview = async () => {
    if (!preview) return;
    setApplying(true);
    setError("");
    try {
      const updated = await app().ApplyInspirationEnrich({
        id: inspirationId,
        title: preview.title,
        genre: preview.genre,
        style: preview.style,
        spark: preview.spark,
        synopsis: preview.synopsis,
        protagonist: preview.protagonist,
        cheat: preview.cheat,
        tags: preview.tags,
      });
      onApplied(updated);
      setPreview(null);
      setDiscussing(false);
      setTurns([]);
      app().ClearInspirationDiscuss().catch(() => {});
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setApplying(false);
    }
  };

  if (preview) {
    return (
      <div className="space-y-4">
        <div className="flex items-center justify-between gap-2">
          <h3 className="text-sm font-medium">确认 AI 生成内容</h3>
          <button
            type="button"
            onClick={() => setPreview(null)}
            className="rounded p-1 text-studio-muted hover:bg-studio-bg hover:text-studio-text"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
        <PreviewField label="标题" value={preview.title} />
        <div className="grid grid-cols-2 gap-3">
          <PreviewField label="题材" value={preview.genre} />
          <PreviewField label="风格" value={preview.style} />
        </div>
        <PreviewField label="标签" value={tagsToInput(preview.tags)} />
        <PreviewField label="主角" value={preview.protagonist} />
        <PreviewField label="金手指" value={preview.cheat} />
        <div>
          <label className="mb-1 block text-xs text-studio-muted">简介</label>
          <div className="max-h-24 overflow-y-auto rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm whitespace-pre-wrap">
            {preview.synopsis || "—"}
          </div>
        </div>
        <div>
          <label className="mb-1 block text-xs text-studio-muted">完整设定（将写入灵感正文）</label>
          <div className="max-h-56 overflow-y-auto rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm whitespace-pre-wrap leading-relaxed">
            {preview.spark}
          </div>
        </div>
        {error && <div className="studio-alert-error-compact">{error}</div>}
        <div className="flex justify-end gap-2">
          <button
            type="button"
            onClick={() => setPreview(null)}
            className="rounded-lg px-4 py-2 text-sm text-studio-muted"
          >
            取消
          </button>
          <button
            type="button"
            onClick={() => void applyPreview()}
            disabled={applying}
            className="inline-flex items-center gap-1 rounded-lg bg-studio-accent px-4 py-2 text-sm text-studio-on-accent disabled:opacity-40"
          >
            {applying ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />}
            应用到灵感
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <p className="text-sm text-studio-muted">
        用 AI 帮你扩写世界观、势力、主角与能力体系。一键润色适合快速丰富；探讨适合慢慢聊细节。
      </p>
      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          onClick={() => void runEnrich()}
          disabled={enriching || loading}
          className="inline-flex items-center gap-1.5 rounded-lg border border-studio-border px-3 py-2 text-sm hover:bg-studio-bg disabled:opacity-40"
        >
          {enriching ? <Loader2 className="h-4 w-4 animate-spin" /> : <Wand2 className="h-4 w-4" />}
          AI 一键润色
        </button>
        {!discussing ? (
          <button
            type="button"
            onClick={() => void startDiscuss()}
            disabled={enriching || loading}
            className="inline-flex items-center gap-1.5 rounded-lg bg-studio-accent/10 px-3 py-2 text-sm text-studio-accent hover:bg-studio-accent/15 disabled:opacity-40"
          >
            {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Sparkles className="h-4 w-4" />}
            开始 AI 探讨
          </button>
        ) : (
          <button
            type="button"
            onClick={() => {
              setDiscussing(false);
              setTurns([]);
              app().ClearInspirationDiscuss().catch(() => {});
            }}
            className="rounded-lg px-3 py-2 text-sm text-studio-muted hover:bg-studio-bg"
          >
            结束探讨
          </button>
        )}
      </div>

      {discussing && (
        <div className="space-y-3">
          <div
            ref={scrollRef}
            className="max-h-64 space-y-3 overflow-y-auto rounded-lg border border-studio-border bg-studio-bg p-3"
          >
            {turns.map((t, i) => (
              <div
                key={i}
                className={`rounded-lg px-3 py-2 text-sm ${
                  t.role === "user" ? "ml-6 bg-studio-panel" : "mr-6 bg-studio-accent/10"
                }`}
              >
                <p className="mb-1 text-[10px] text-studio-muted">{t.role === "user" ? "你" : "顾问"}</p>
                <p className="whitespace-pre-wrap leading-relaxed">{t.content}</p>
              </div>
            ))}
            {streaming && (
              <div className="mr-6 rounded-lg bg-studio-accent/10 px-3 py-2 text-sm">
                <p className="mb-1 text-[10px] text-studio-muted">顾问</p>
                <p className="whitespace-pre-wrap leading-relaxed">{streaming}</p>
              </div>
            )}
            {loading && !streaming && (
              <div className="flex items-center gap-2 text-xs text-studio-muted">
                <Loader2 className="h-3 w-3 animate-spin" />
                思考中…
              </div>
            )}
          </div>
          <div className="flex gap-2">
            <input
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder="说说你想完善的方向…"
              disabled={loading}
              onKeyDown={(e) => e.key === "Enter" && !e.shiftKey && void send()}
              className="min-w-0 flex-1 rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none disabled:opacity-50"
            />
            <button
              type="button"
              onClick={() => void send()}
              disabled={loading || !input.trim()}
              className="rounded-lg bg-studio-accent px-3 text-studio-on-accent disabled:opacity-40"
            >
              <Send className="h-4 w-4" />
            </button>
          </div>
          <div className="flex justify-end">
            <button
              type="button"
              onClick={() => void finishDiscuss()}
              disabled={loading || turns.length < 2}
              className="inline-flex items-center gap-1 rounded-lg border border-studio-border px-3 py-1.5 text-xs hover:bg-studio-bg disabled:opacity-40"
            >
              <Wand2 className="h-3.5 w-3.5" />
              完成探讨，生成设定
            </button>
          </div>
        </div>
      )}

      {error && <div className="studio-alert-error-compact">{error}</div>}
    </div>
  );
}

function PreviewField({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <label className="mb-1 block text-xs text-studio-muted">{label}</label>
      <div className="rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm">{value || "—"}</div>
    </div>
  );
}
