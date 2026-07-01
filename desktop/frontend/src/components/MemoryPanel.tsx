import { useCallback, useEffect, useState } from "react";
import { AlertTriangle, Archive, Check, Loader2, Pencil, Plus, X } from "lucide-react";
import { CreateMemoryInput, ForeshadowDTO, MemoryConflictDTO, MemoryDTO, app } from "../lib/wails";

export type MemoryFocus = "memories" | "foreshadows" | "resolved" | "conflicts";

const memoryCategories = [
  { value: "character", label: "角色" },
  { value: "world", label: "世界观" },
  { value: "plot", label: "剧情" },
  { value: "style", label: "写法" },
];

type Props = {
  focus: MemoryFocus;
  onFocusChange?: (focus: MemoryFocus) => void;
  highlightId?: string;
};

const focusTabs: { id: MemoryFocus; label: string }[] = [
  { id: "memories", label: "长期记忆" },
  { id: "foreshadows", label: "Open 伏笔" },
  { id: "resolved", label: "已回收" },
  { id: "conflicts", label: "冲突检测" },
];

export default function MemoryPanel({ focus, onFocusChange, highlightId = "" }: Props) {
  const [memories, setMemories] = useState<MemoryDTO[]>([]);
  const [foreshadows, setForeshadows] = useState<ForeshadowDTO[]>([]);
  const [conflicts, setConflicts] = useState<MemoryConflictDTO[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [editingMemory, setEditingMemory] = useState<MemoryDTO | null>(null);
  const [editingForeshadow, setEditingForeshadow] = useState<ForeshadowDTO | null>(null);
  const [creating, setCreating] = useState(false);
  const [newMemory, setNewMemory] = useState<CreateMemoryInput>({
    category: "plot",
    subject: "",
    content: "",
    source_chapter: 0,
  });
  const [saving, setSaving] = useState(false);
  const [resolveID, setResolveID] = useState("");
  const [resolveChapter, setResolveChapter] = useState(0);

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      if (focus === "conflicts") {
        const list = await app().FindMemoryConflicts();
        setConflicts(list ?? []);
        return;
      }
      const fsStatus = focus === "resolved" ? "resolved" : focus === "foreshadows" ? "open" : "";
      const [mem, fs] = await Promise.all([
        focus === "memories" ? app().ListMemories() : Promise.resolve([] as MemoryDTO[]),
        focus === "foreshadows" || focus === "resolved"
          ? app().ListForeshadows(fsStatus)
          : Promise.resolve([] as ForeshadowDTO[]),
      ]);
      if (focus === "memories") {
        setMemories(mem.filter((m) => m.status !== "archived"));
      }
      setForeshadows(fs);
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
      document.getElementById(`memory-item-${highlightId}`)?.scrollIntoView({ block: "center", behavior: "smooth" });
    }, 100);
    return () => clearTimeout(timer);
  }, [highlightId, memories, foreshadows, focus]);

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

  const saveMemory = async () => {
    if (!editingMemory) return;
    setSaving(true);
    try {
      await app().UpdateMemory({
        id: editingMemory.id,
        category: editingMemory.category,
        subject: editingMemory.subject,
        content: editingMemory.content,
        source_chapter: editingMemory.source_chapter,
      });
      setEditingMemory(null);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  };

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

  const createMemory = async () => {
    if (!newMemory.subject.trim() || !newMemory.content.trim()) {
      setError("主题和内容不能为空");
      return;
    }
    setSaving(true);
    try {
      await app().CreateMemory(newMemory);
      setCreating(false);
      setNewMemory({ category: "plot", subject: "", content: "", source_chapter: 0 });
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  };

  const archiveMemory = async (id: string) => {
    if (!confirm("归档此记忆？（不会删除，仅不再注入写章上下文）")) return;
    try {
      await app().ArchiveMemory(id);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
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

  if (loading) {
    return (
      <div className="flex items-center gap-2 text-sm text-studio-muted">
        <Loader2 className="h-4 w-4 animate-spin" />
        加载中…
      </div>
    );
  }

  if (focus === "conflicts") {
    return (
      <div className="min-h-0 flex-1 overflow-y-auto">
        {tabBar}
        <h2 className="mb-2 text-sm font-medium text-studio-muted">记忆冲突 ({conflicts.length})</h2>
        <p className="mb-4 text-xs leading-relaxed text-studio-muted">
          同一主题（subject）下存在多条活跃记忆时，写章上下文可能互相矛盾。建议合并或归档冗余条目。
        </p>
        {error && <div className="mb-4 studio-alert-error-compact">{error}</div>}
        {conflicts.length === 0 ? (
          <div className="flex flex-col items-center rounded-xl border border-dashed border-studio-border py-10 text-center">
            <Check className="mb-2 h-8 w-8 text-[rgb(var(--studio-diff-add-stat))]" />
            <p className="text-sm text-studio-text">未发现 subject 冲突</p>
          </div>
        ) : (
          <ul className="space-y-4">
            {conflicts.map((c) => (
              <li key={c.subject} className="rounded-xl border border-[rgb(var(--studio-warning-border))] bg-[rgb(var(--studio-warning-bg))] p-4">
                <div className="mb-3 flex items-center gap-2">
                  <AlertTriangle className="h-4 w-4 text-[rgb(var(--studio-warning-fg))]" />
                  <span className="text-sm font-medium text-studio-text">
                    「{c.subject}」· {c.count} 条记忆
                  </span>
                </div>
                <ul className="space-y-2">
                  {c.memories.map((m) => (
                    <li
                      key={m.id}
                      id={`memory-item-${m.id}`}
                      className="rounded-lg border border-studio-border bg-studio-panel p-3"
                    >
                      <div className="flex flex-wrap items-center gap-2 text-xs text-studio-muted">
                        <span className="rounded bg-studio-border px-2 py-0.5">{m.category}</span>
                        {m.source_chapter > 0 && <span>第 {m.source_chapter} 章</span>}
                      </div>
                      <p className="mt-2 text-sm leading-relaxed">{m.content}</p>
                    </li>
                  ))}
                </ul>
              </li>
            ))}
          </ul>
        )}
      </div>
    );
  }

  if (focus === "foreshadows" || focus === "resolved") {
    const isOpen = focus === "foreshadows";
    const title = isOpen ? "Open 伏笔" : "已回收伏笔";
    return (
      <div className="min-h-0 flex-1 overflow-y-auto">
        {tabBar}
        <h2 className="mb-4 text-sm font-medium text-studio-muted">
          {title} ({foreshadows.length})
        </h2>
        {error && <div className="mb-4 studio-alert-error-compact">{error}</div>}
        {foreshadows.length === 0 ? (
          <p className="text-sm text-studio-muted">
            {isOpen ? "暂无 open 伏笔" : "暂无已回收伏笔"}
          </p>
        ) : (
          <ul className="space-y-3">
            {foreshadows.map((f) => (
              <li
                key={f.id}
                id={`memory-item-${f.id}`}
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
                      onChange={(e) =>
                        setEditingForeshadow({ ...editingForeshadow, description: e.target.value })
                      }
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
                      <button
                        type="button"
                        onClick={() => setEditingForeshadow(null)}
                        className="text-xs text-studio-muted"
                      >
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

  return (
    <div className="min-h-0 flex-1 overflow-y-auto">
      {tabBar}
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-sm font-medium text-studio-muted">长期记忆 ({memories.length})</h2>
        <button
          type="button"
          onClick={() => setCreating(true)}
          className="inline-flex items-center gap-1 rounded-lg border border-studio-border px-2 py-1 text-xs text-studio-muted hover:text-studio-text"
        >
          <Plus className="h-3.5 w-3.5" />
          新增
        </button>
      </div>

      {error && <div className="mb-4 studio-alert-error-compact">{error}</div>}

      {creating && (
        <div className="mb-4 rounded-xl border border-studio-accent/30 bg-studio-panel p-4">
          <div className="mb-3 flex items-center justify-between">
            <span className="text-sm font-medium">新建记忆</span>
            <button type="button" onClick={() => setCreating(false)}>
              <X className="h-4 w-4 text-studio-muted" />
            </button>
          </div>
          <div className="space-y-2">
            <select
              value={newMemory.category}
              onChange={(e) => setNewMemory((m) => ({ ...m, category: e.target.value }))}
              className="w-full rounded border border-studio-border bg-studio-bg px-2 py-1.5 text-sm"
            >
              {memoryCategories.map((c) => (
                <option key={c.value} value={c.value}>
                  {c.label}
                </option>
              ))}
            </select>
            <input
              value={newMemory.subject}
              onChange={(e) => setNewMemory((m) => ({ ...m, subject: e.target.value }))}
              placeholder="主题"
              className="w-full rounded border border-studio-border bg-studio-bg px-2 py-1.5 text-sm outline-none"
            />
            <textarea
              value={newMemory.content}
              onChange={(e) => setNewMemory((m) => ({ ...m, content: e.target.value }))}
              placeholder="内容"
              rows={3}
              className="w-full resize-none rounded border border-studio-border bg-studio-bg px-2 py-1.5 text-sm outline-none"
            />
            <button
              type="button"
              onClick={createMemory}
              disabled={saving}
              className="rounded-lg bg-studio-accent px-3 py-1.5 text-xs text-studio-on-accent disabled:opacity-40"
            >
              {saving ? "保存中…" : "保存"}
            </button>
          </div>
        </div>
      )}

      {memories.length === 0 ? (
        <p className="text-sm text-studio-muted">暂无记忆条目</p>
      ) : (
        <ul className="space-y-3">
          {memories.map((m) => (
            <li
              key={m.id}
              id={`memory-item-${m.id}`}
              className={`rounded-xl border bg-studio-panel p-4 ${
                highlightId === m.id
                  ? "border-studio-accent ring-2 ring-studio-accent/30"
                  : "border-studio-border"
              }`}
            >
              {editingMemory?.id === m.id ? (
                <div className="space-y-2">
                  <select
                    value={editingMemory.category}
                    onChange={(e) => setEditingMemory({ ...editingMemory, category: e.target.value })}
                    className="w-full rounded border border-studio-border bg-studio-bg px-2 py-1.5 text-sm"
                  >
                    {memoryCategories.map((c) => (
                      <option key={c.value} value={c.value}>
                        {c.label}
                      </option>
                    ))}
                  </select>
                  <input
                    value={editingMemory.subject}
                    onChange={(e) => setEditingMemory({ ...editingMemory, subject: e.target.value })}
                    className="w-full rounded border border-studio-border bg-studio-bg px-2 py-1.5 text-sm outline-none"
                  />
                  <textarea
                    value={editingMemory.content}
                    onChange={(e) => setEditingMemory({ ...editingMemory, content: e.target.value })}
                    rows={4}
                    className="w-full resize-none rounded border border-studio-border bg-studio-bg px-2 py-1.5 text-sm outline-none"
                  />
                  <div className="flex gap-2">
                    <button
                      type="button"
                      onClick={saveMemory}
                      disabled={saving}
                      className="rounded-lg bg-studio-accent px-3 py-1 text-xs text-studio-on-accent disabled:opacity-40"
                    >
                      {saving ? <Loader2 className="h-3 w-3 animate-spin" /> : "保存"}
                    </button>
                    <button type="button" onClick={() => setEditingMemory(null)} className="text-xs text-studio-muted">
                      取消
                    </button>
                  </div>
                </div>
              ) : (
                <>
                  <div className="flex flex-wrap items-center gap-2 text-xs text-studio-muted">
                    <span className="rounded bg-studio-border px-2 py-0.5">{m.category}</span>
                    {m.subject && <span>{m.subject}</span>}
                    {m.source_chapter > 0 && <span>第 {m.source_chapter} 章</span>}
                    <div className="ml-auto flex gap-1">
                      <button
                        type="button"
                        onClick={() => setEditingMemory({ ...m })}
                        className="rounded p-1 hover:bg-studio-bg hover:text-studio-text"
                        title="编辑"
                      >
                        <Pencil className="h-3.5 w-3.5" />
                      </button>
                      <button
                        type="button"
                        onClick={() => archiveMemory(m.id)}
                        className="rounded p-1 hover:bg-studio-bg hover:text-studio-text"
                        title="归档"
                      >
                        <Archive className="h-3.5 w-3.5" />
                      </button>
                    </div>
                  </div>
                  <p className="mt-2 text-sm leading-relaxed">{m.content}</p>
                </>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
