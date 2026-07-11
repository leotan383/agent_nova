import { useEffect, useState } from "react";
import { Loader2, Trash2 } from "lucide-react";
import {
  InspirationDTO,
  app,
  genreOptions,
  styleOptions,
} from "../lib/wails";
import { inspirationStatusLabel, parseTagsInput, tagsToInput } from "../lib/inspirationUtils";
import InspirationAIPanel from "./InspirationAIPanel";

type Props = {
  id: string;
  onClose: () => void;
  onCreateNovel: () => void;
  onSaved: () => void;
  onDeleted: () => void;
};

type Tab = "spark" | "init" | "ai";

export default function InspirationEditor({ id, onClose, onCreateNovel, onSaved, onDeleted }: Props) {
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [tab, setTab] = useState<Tab>("spark");
  const [data, setData] = useState<InspirationDTO | null>(null);
  const [title, setTitle] = useState("");
  const [spark, setSpark] = useState("");
  const [genre, setGenre] = useState("");
  const [style, setStyle] = useState("");
  const [synopsis, setSynopsis] = useState("");
  const [protagonist, setProtagonist] = useState("");
  const [cheat, setCheat] = useState("");
  const [tagsInput, setTagsInput] = useState("");

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setLoading(true);
      setError("");
      try {
        const insp = await app().GetInspiration(id);
        if (cancelled) return;
        setData(insp);
        setTitle(insp.title);
        setSpark(insp.spark);
        setGenre(insp.genre || "");
        setStyle(insp.style || "");
        setSynopsis(insp.synopsis || "");
        setProtagonist(insp.protagonist || "");
        setCheat(insp.cheat || "");
        setTagsInput(tagsToInput(insp.tags));
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e));
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [id]);

  const save = async () => {
    if (!spark.trim()) {
      setError("灵感正文不能为空");
      return;
    }
    setSaving(true);
    setError("");
    try {
      const updated = await app().UpdateInspiration({
        id,
        title: title.trim(),
        spark: spark.trim(),
        genre,
        style,
        synopsis,
        protagonist,
        cheat,
        tags: parseTagsInput(tagsInput),
      });
      setData(updated);
      onSaved();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  };

  const applyAIPreview = (updated: InspirationDTO) => {
    setData(updated);
    setTitle(updated.title);
    setSpark(updated.spark);
    setGenre(updated.genre || "");
    setStyle(updated.style || "");
    setSynopsis(updated.synopsis || "");
    setProtagonist(updated.protagonist || "");
    setCheat(updated.cheat || "");
    setTagsInput(tagsToInput(updated.tags));
    setTab("spark");
    onSaved();
  };

  const remove = async () => {
    if (!window.confirm("确定删除这条灵感？此操作不可恢复。")) return;
    setSaving(true);
    try {
      await app().DeleteInspiration(id);
      onDeleted();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-16 text-sm text-studio-muted">
        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
        加载中…
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h2 className="text-lg font-medium">{data?.display_title || "编辑灵感"}</h2>
          {data && (
            <p className="mt-0.5 text-xs text-studio-muted">
              {inspirationStatusLabel[data.status] || data.status}
              {data.novel_title ? ` · 已用于《${data.novel_title}》` : ""}
            </p>
          )}
        </div>
        <button type="button" onClick={onClose} className="text-sm text-studio-muted hover:text-studio-text">
          关闭
        </button>
      </div>

      <div className="flex gap-1 rounded-lg border border-studio-border bg-studio-bg p-1">
        <button
          type="button"
          onClick={() => setTab("spark")}
          className={`flex-1 rounded-md px-3 py-1.5 text-xs ${
            tab === "spark" ? "bg-studio-panel text-studio-accent shadow-sm" : "text-studio-muted"
          }`}
        >
          灵感正文
        </button>
        <button
          type="button"
          onClick={() => setTab("init")}
          className={`flex-1 rounded-md px-3 py-1.5 text-xs ${
            tab === "init" ? "bg-studio-panel text-studio-accent shadow-sm" : "text-studio-muted"
          }`}
        >
          立项信息
        </button>
        <button
          type="button"
          onClick={() => setTab("ai")}
          className={`flex-1 rounded-md px-3 py-1.5 text-xs ${
            tab === "ai" ? "bg-studio-panel text-studio-accent shadow-sm" : "text-studio-muted"
          }`}
        >
          AI 丰富
        </button>
      </div>

      {tab === "ai" ? (
        <InspirationAIPanel inspirationId={id} onApplied={applyAIPreview} />
      ) : tab === "spark" ? (
        <div className="space-y-3">
          <div>
            <label className="mb-1 block text-xs text-studio-muted">短标题（可选）</label>
            <input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="给灵感起个名字"
              className="w-full rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none focus:border-studio-accent"
            />
          </div>
          <div>
            <label className="mb-1 block text-xs text-studio-muted">核心想法 *</label>
            <textarea
              value={spark}
              onChange={(e) => setSpark(e.target.value)}
              rows={8}
              className="w-full resize-none rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none focus:border-studio-accent"
            />
          </div>
          <div>
            <label className="mb-1 block text-xs text-studio-muted">标签（用顿号或逗号分隔）</label>
            <input
              value={tagsInput}
              onChange={(e) => setTagsInput(e.target.value)}
              placeholder="赛博、末世、女频"
              className="w-full rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none focus:border-studio-accent"
            />
          </div>
        </div>
      ) : (
        <div className="space-y-3">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="mb-1 block text-xs text-studio-muted">题材</label>
              <select
                value={genre}
                onChange={(e) => setGenre(e.target.value)}
                className="w-full rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm"
              >
                <option value="">暂不指定</option>
                {genreOptions.map((g) => (
                  <option key={g} value={g}>
                    {g}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="mb-1 block text-xs text-studio-muted">风格</label>
              <select
                value={style}
                onChange={(e) => setStyle(e.target.value)}
                className="w-full rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm"
              >
                <option value="">暂不指定</option>
                {styleOptions.map((s) => (
                  <option key={s} value={s}>
                    {s}
                  </option>
                ))}
              </select>
            </div>
          </div>
          <div>
            <label className="mb-1 block text-xs text-studio-muted">简介</label>
            <textarea
              value={synopsis}
              onChange={(e) => setSynopsis(e.target.value)}
              rows={3}
              className="w-full resize-none rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none focus:border-studio-accent"
            />
          </div>
          <div>
            <label className="mb-1 block text-xs text-studio-muted">主角</label>
            <input
              value={protagonist}
              onChange={(e) => setProtagonist(e.target.value)}
              className="w-full rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none focus:border-studio-accent"
            />
          </div>
          <div>
            <label className="mb-1 block text-xs text-studio-muted">金手指</label>
            <input
              value={cheat}
              onChange={(e) => setCheat(e.target.value)}
              className="w-full rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none focus:border-studio-accent"
            />
          </div>
        </div>
      )}

      {tab !== "ai" && error && <div className="studio-alert-error-compact">{error}</div>}

      {tab !== "ai" && (
      <div className="flex flex-wrap items-center justify-between gap-2">
        <button
          type="button"
          onClick={() => void remove()}
          disabled={saving}
          className="inline-flex items-center gap-1 text-sm text-[rgb(var(--studio-danger-fg))] hover:underline disabled:opacity-40"
        >
          <Trash2 className="h-3.5 w-3.5" />
          删除
        </button>
        <div className="flex gap-2">
          {data?.status !== "used" && (
            <button
              type="button"
              onClick={onCreateNovel}
              className="rounded-lg border border-studio-border px-4 py-2 text-sm hover:bg-studio-bg"
            >
              创建小说
            </button>
          )}
          <button
            type="button"
            onClick={() => void save()}
            disabled={saving || !spark.trim()}
            className="inline-flex items-center gap-1 rounded-lg bg-studio-accent px-4 py-2 text-sm font-medium text-studio-on-accent disabled:opacity-40"
          >
            {saving && <Loader2 className="h-4 w-4 animate-spin" />}
            保存
          </button>
        </div>
      </div>
      )}
    </div>
  );
}
