import { useEffect, useRef, useState } from "react";
import {
  Check,
  Loader2,
  Send,
  Sparkles,
  Wand2,
} from "lucide-react";
import { DISCOVER_EVENTS, eventsOn } from "../lib/runtime";
import {
  CoachTurnDTO,
  DiscoverPreviewDTO,
  app,
  genreOptions,
} from "../lib/wails";

type Props = {
  onCreated: () => void;
  onCancel: () => void;
  inspirationId?: string;
  seedPrompt?: string;
  initialGenre?: string;
};

export default function DiscoverCreatePanel({
  onCreated,
  onCancel,
  inspirationId = "",
  seedPrompt = "",
  initialGenre = "玄幻",
}: Props) {
  const [genre, setGenre] = useState(initialGenre || "玄幻");
  const [started, setStarted] = useState(false);
  const [turns, setTurns] = useState<CoachTurnDTO[]>([]);
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);
  const [streaming, setStreaming] = useState("");
  const [error, setError] = useState("");
  const [preview, setPreview] = useState<DiscoverPreviewDTO | null>(null);
  const [dir, setDir] = useState("");
  const [enrich, setEnrich] = useState(true);
  const [creating, setCreating] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight });
  }, [turns, streaming]);

  useEffect(() => {
    if (initialGenre) setGenre(initialGenre);
  }, [initialGenre]);

  useEffect(() => {
    const unsubs = [
      eventsOn(DISCOVER_EVENTS.stream, (p) => {
        setStreaming((t) => t + (p.delta || ""));
      }),
      eventsOn(DISCOVER_EVENTS.done, (p) => {
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
      eventsOn(DISCOVER_EVENTS.error, (p) => {
        setError(p.error || "探讨失败");
        setStreaming("");
        setLoading(false);
      }),
    ];
    return () => {
      unsubs.forEach((u) => u());
      app().ClearDiscover().catch(() => {});
    };
  }, []);

  const start = async () => {
    setError("");
    setLoading(true);
    setStreaming("");
    try {
      const t = await app().StartDiscover(genre, seedPrompt || "");
      setTurns(t ?? []);
      setStarted(true);
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
      await app().SendDiscoverMessage(msg);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setLoading(false);
    }
  };

  const finish = async () => {
    setError("");
    setLoading(true);
    try {
      const p = await app().FinishDiscover();
      setPreview(p);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  };

  const create = async () => {
    if (!preview || !dir.trim()) {
      setError("请填写保存目录");
      return;
    }
    setCreating(true);
    setError("");
    try {
      await app().CreateNovelFromDiscover({
        dir: dir.trim(),
        title: preview.title,
        genre: preview.genre,
        style: preview.style || "热血",
        target_words: 300000,
        chapter_words: 4000,
        protagonist: preview.protagonist,
        cheat: preview.cheat,
        synopsis: preview.synopsis,
        enrich,
        inspiration_id: inspirationId || undefined,
      });
      onCreated();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setCreating(false);
    }
  };

  if (preview) {
    return (
      <div className="space-y-4">
        <h3 className="text-sm font-medium">确认立项信息</h3>
        <div className="grid gap-3 sm:grid-cols-2">
          <Field label="书名" value={preview.title} onChange={(v) => setPreview({ ...preview, title: v })} />
          <Field label="题材" value={preview.genre} onChange={(v) => setPreview({ ...preview, genre: v })} />
          <Field label="风格" value={preview.style} onChange={(v) => setPreview({ ...preview, style: v })} />
          <Field label="主角" value={preview.protagonist} onChange={(v) => setPreview({ ...preview, protagonist: v })} />
        </div>
        <Field label="金手指" value={preview.cheat} onChange={(v) => setPreview({ ...preview, cheat: v })} />
        <div>
          <label className="mb-1 block text-xs text-studio-muted">故事核心</label>
          <textarea
            value={preview.synopsis}
            onChange={(e) => setPreview({ ...preview, synopsis: e.target.value })}
            rows={3}
            className="w-full resize-none rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none"
          />
        </div>
        {preview.pitch && (
          <p className="text-xs text-studio-muted">梗概：{preview.pitch}</p>
        )}
        <div>
          <label className="mb-1 block text-xs text-studio-muted">保存目录</label>
          <div className="flex gap-2">
            <input
              value={dir}
              onChange={(e) => setDir(e.target.value)}
              className="flex-1 rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none"
            />
            <button
              type="button"
              onClick={async () => {
                const p = await app().PickCreateDirectory();
                if (p) setDir(p);
              }}
              className="shrink-0 rounded-lg border border-studio-border px-3 text-sm"
            >
              选择
            </button>
          </div>
        </div>
        <label className="flex items-center gap-2 text-sm text-studio-muted">
          <input type="checkbox" checked={enrich} onChange={(e) => setEnrich(e.target.checked)} />
          AI 完善设定集与总纲（需 API，耗时 1-2 分钟）
        </label>
        {error && <div className="studio-alert-error-compact">{error}</div>}
        <div className="flex justify-end gap-2">
          <button type="button" onClick={() => setPreview(null)} className="rounded-lg px-4 py-2 text-sm text-studio-muted">
            返回探讨
          </button>
          <button
            type="button"
            onClick={create}
            disabled={creating || !dir.trim()}
            className="inline-flex items-center gap-1 rounded-lg bg-studio-accent px-4 py-2 text-sm text-studio-on-accent disabled:opacity-40"
          >
            {creating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />}
            创建项目
          </button>
        </div>
      </div>
    );
  }

  if (!started) {
    return (
      <div className="space-y-4">
        <p className="text-sm text-studio-muted">
          与 AI 顾问聊聊你想写的故事，聊够了再提炼书名、主角、金手指并创建项目。
        </p>
        <div>
          <label className="mb-1 block text-xs text-studio-muted">初步题材（可选）</label>
          <select
            value={genre}
            onChange={(e) => setGenre(e.target.value)}
            className="w-full rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm"
          >
            {genreOptions.map((g) => (
              <option key={g} value={g}>
                {g}
              </option>
            ))}
          </select>
        </div>
        {error && <div className="studio-alert-error-compact">{error}</div>}
        <div className="flex justify-end gap-2">
          <button type="button" onClick={onCancel} className="rounded-lg px-4 py-2 text-sm text-studio-muted">
            取消
          </button>
          <button
            type="button"
            onClick={start}
            disabled={loading}
            className="inline-flex items-center gap-1 rounded-lg bg-studio-accent px-4 py-2 text-sm text-studio-on-accent disabled:opacity-40"
          >
            {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Sparkles className="h-4 w-4" />}
            开始探讨
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex max-h-[60vh] flex-col gap-3">
      <div ref={scrollRef} className="min-h-[240px] flex-1 space-y-3 overflow-y-auto rounded-lg border border-studio-border bg-studio-bg p-3">
        {turns.map((t, i) => (
          <div
            key={i}
            className={`rounded-lg px-3 py-2 text-sm ${
              t.role === "user" ? "ml-8 bg-studio-panel" : "mr-8 bg-studio-accent/10"
            }`}
          >
            <p className="mb-1 text-[10px] text-studio-muted">{t.role === "user" ? "你" : "顾问"}</p>
            <p className="whitespace-pre-wrap leading-relaxed">{t.content}</p>
          </div>
        ))}
        {streaming && (
          <div className="mr-8 rounded-lg bg-studio-accent/10 px-3 py-2 text-sm">
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
          placeholder="说说你的想法…"
          disabled={loading}
          onKeyDown={(e) => e.key === "Enter" && !e.shiftKey && send()}
          className="min-w-0 flex-1 rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none disabled:opacity-50"
        />
        <button
          type="button"
          onClick={send}
          disabled={loading || !input.trim()}
          className="rounded-lg bg-studio-accent px-3 text-studio-on-accent disabled:opacity-40"
        >
          <Send className="h-4 w-4" />
        </button>
      </div>

      {error && <div className="studio-alert-error-compact">{error}</div>}

      <div className="flex flex-wrap justify-between gap-2">
        <button type="button" onClick={onCancel} className="rounded-lg px-3 py-1.5 text-xs text-studio-muted">
          取消
        </button>
        <div className="flex gap-2">
          <button
            type="button"
            onClick={finish}
            disabled={loading || turns.length < 2}
            className="inline-flex items-center gap-1 rounded-lg border border-studio-border px-3 py-1.5 text-xs hover:bg-studio-bg"
          >
            <Wand2 className="h-3.5 w-3.5" />
            完成探讨
          </button>
        </div>
      </div>
    </div>
  );
}

function Field({
  label,
  value,
  onChange,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
}) {
  return (
    <div>
      <label className="mb-1 block text-xs text-studio-muted">{label}</label>
      <input
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none"
      />
    </div>
  );
}
