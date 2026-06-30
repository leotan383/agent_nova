import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { FolderOpen, Plus, RefreshCw, Settings } from "lucide-react";
import ThemeToggle from "../components/ThemeToggle";
import DiscoverCreatePanel from "../components/DiscoverCreatePanel";
import SettingsDialog from "../components/SettingsDialog";
import NovelCardView from "../components/NovelCard";
import {
  NovelCard,
  app,
  chapterWordOptions,
  genreOptions,
  styleOptions,
  targetWordOptions,
} from "../lib/wails";

const defaultCreateForm = () => ({
  dir: "",
  title: "",
  genre: "玄幻",
  style: "热血",
  targetWords: 300000,
  chapterWords: 4000,
  synopsis: "",
});

export default function LibraryPage() {
  const navigate = useNavigate();
  const [novels, setNovels] = useState<NovelCard[]>([]);
  const [activeId, setActiveId] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const [createMode, setCreateMode] = useState<"form" | "discover">("discover");
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [form, setForm] = useState(defaultCreateForm);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [list, active] = await Promise.all([
        app().ListNovels(false),
        app().GetActiveNovel(),
      ]);
      setNovels(list);
      setActiveId(active.id || "");
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const openNovel = async (id: string) => {
    try {
      await app().SwitchNovel(id);
      navigate("/studio");
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      await refresh();
    }
  };

  const handleOpenExisting = async () => {
    const path = await app().PickNovelDirectory();
    if (!path) return;
    await app().RegisterNovel(path);
    await refresh();
  };

  const closeCreate = () => {
    setShowCreate(false);
    setCreateMode("discover");
    setForm(defaultCreateForm());
    app().ClearDiscover().catch(() => {});
  };

  const handleCreate = async () => {
    if (!form.dir.trim()) {
      setError("请选择保存目录");
      return;
    }
    if (!form.title.trim()) {
      setError("书名不能为空");
      return;
    }
    setCreating(true);
    setError("");
    try {
      await app().CreateNovel({
        dir: form.dir,
        title: form.title.trim(),
        genre: form.genre,
        style: form.style,
        target_words: form.targetWords,
        chapter_words: form.chapterWords,
        synopsis: form.synopsis.trim(),
      });
      closeCreate();
      await refresh();
      navigate("/studio");
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setCreating(false);
    }
  };

  const canCreate = form.dir.trim() !== "" && form.title.trim() !== "";

  return (
    <div className="h-full overflow-y-auto">
      <header className="flex items-center justify-between border-b border-studio-border px-8 py-5">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Nova Studio</h1>
          <p className="mt-1 text-sm text-studio-muted">小说书库 · 选择或创建你的故事</p>
        </div>
        <div className="flex items-center gap-3">
          <ThemeToggle />
          <button
            type="button"
            onClick={() => setSettingsOpen(true)}
            className="inline-flex items-center gap-2 rounded-lg border border-studio-border px-3 py-2 text-sm text-studio-muted hover:text-studio-text"
          >
            <Settings className="h-4 w-4" />
            设置
          </button>
          <button
            type="button"
            onClick={refresh}
            className="inline-flex items-center gap-2 rounded-lg border border-studio-border px-3 py-2 text-sm text-studio-muted hover:text-studio-text"
          >
            <RefreshCw className="h-4 w-4" />
            刷新
          </button>
          <button
            type="button"
            onClick={handleOpenExisting}
            className="inline-flex items-center gap-2 rounded-lg border border-studio-border px-4 py-2 text-sm hover:border-studio-muted"
          >
            <FolderOpen className="h-4 w-4" />
            打开已有
          </button>
          <button
            type="button"
            onClick={() => setShowCreate(true)}
            className="inline-flex items-center gap-2 rounded-lg bg-studio-accent px-4 py-2 text-sm font-medium text-studio-on-accent hover:brightness-110"
          >
            <Plus className="h-4 w-4" />
            新建小说
          </button>
        </div>
      </header>

      <main className="px-8 py-8">
        {error && !showCreate && (
          <div className="mb-6 studio-alert-error">
            {error}
          </div>
        )}

        {loading ? (
          <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            {[1, 2, 3].map((i) => (
              <div
                key={i}
                className="h-64 animate-pulse rounded-xl border border-studio-border bg-studio-panel"
              />
            ))}
          </div>
        ) : novels.length === 0 ? (
          <div className="flex flex-col items-center justify-center rounded-2xl border border-dashed border-studio-border py-24 text-center">
            <p className="text-lg text-studio-text">还没有小说</p>
            <p className="mt-2 max-w-md text-sm text-studio-muted">
              从模糊想法开始新建，或打开磁盘上已有的 nova 项目（含 nova.yaml 的目录）
            </p>
            <div className="mt-8 flex gap-3">
              <button
                type="button"
                onClick={() => setShowCreate(true)}
                className="rounded-lg bg-studio-accent px-5 py-2.5 text-sm font-medium text-studio-on-accent"
              >
                新建第一本
              </button>
              <button
                type="button"
                onClick={handleOpenExisting}
                className="rounded-lg border border-studio-border px-5 py-2.5 text-sm"
              >
                打开已有项目
              </button>
            </div>
          </div>
        ) : (
          <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            {novels.map((n) => (
              <NovelCardView
                key={n.id}
                novel={n}
                active={n.id === activeId}
                onOpen={() => openNovel(n.id)}
                onReveal={() => app().RevealInFolder(n.path)}
                onRemove={async () => {
                  if (confirm(`从书库移除「${n.title}」？（不会删除文件）`)) {
                    await app().RemoveFromLibrary(n.id);
                    await refresh();
                  }
                }}
                onTogglePin={async () => {
                  await app().SetNovelPinned(n.id, !n.pinned);
                  await refresh();
                }}
              />
            ))}
            <button
              type="button"
              onClick={() => setShowCreate(true)}
              className="flex min-h-[16rem] flex-col items-center justify-center rounded-xl border border-dashed border-studio-border text-studio-muted transition hover:border-studio-accent/50 hover:text-studio-accent"
            >
              <Plus className="mb-2 h-8 w-8" />
              新建小说
            </button>
          </div>
        )}
      </main>

      {showCreate && (
        <div className="studio-modal-overlay fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-2xl border border-studio-border bg-studio-panel p-6 shadow-card">
            <h2 className="text-lg font-medium">新建小说</h2>
            <div className="mt-4 flex gap-1 rounded-lg border border-studio-border bg-studio-bg p-1">
              <button
                type="button"
                onClick={() => setCreateMode("discover")}
                className={`flex-1 rounded-md px-3 py-1.5 text-xs ${
                  createMode === "discover" ? "bg-studio-panel text-studio-accent shadow-sm" : "text-studio-muted"
                }`}
              >
                AI 探讨立项
              </button>
              <button
                type="button"
                onClick={() => setCreateMode("form")}
                className={`flex-1 rounded-md px-3 py-1.5 text-xs ${
                  createMode === "form" ? "bg-studio-panel text-studio-accent shadow-sm" : "text-studio-muted"
                }`}
              >
                表单快速创建
              </button>
            </div>

            {createMode === "discover" ? (
              <div className="mt-5">
                <DiscoverCreatePanel
                  onCreated={async () => {
                    closeCreate();
                    await refresh();
                    navigate("/studio");
                  }}
                  onCancel={closeCreate}
                />
              </div>
            ) : (
              <>
            <p className="mt-1 text-sm text-studio-muted">填写基本信息，创建项目骨架</p>

            {error && (
              <div className="mt-4 studio-alert-error-compact">
                {error}
              </div>
            )}

            <div className="mt-5 space-y-4">
              <div>
                <label className="mb-1 block text-xs text-studio-muted">
                  书名 <span className="text-[rgb(var(--studio-diff-del-stat))]">*</span>
                </label>
                <input
                  value={form.title}
                  onChange={(e) => setForm((f) => ({ ...f, title: e.target.value }))}
                  placeholder="例如：剑出长安"
                  className="w-full rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none focus:border-studio-accent"
                />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="mb-1 block text-xs text-studio-muted">题材</label>
                  <select
                    value={form.genre}
                    onChange={(e) => setForm((f) => ({ ...f, genre: e.target.value }))}
                    className="w-full rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none"
                  >
                    {genreOptions.map((g) => (
                      <option key={g} value={g}>
                        {g}
                      </option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="mb-1 block text-xs text-studio-muted">写作风格</label>
                  <select
                    value={form.style}
                    onChange={(e) => setForm((f) => ({ ...f, style: e.target.value }))}
                    className="w-full rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none"
                  >
                    {styleOptions.map((s) => (
                      <option key={s} value={s}>
                        {s}
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="mb-1 block text-xs text-studio-muted">目标字数</label>
                  <select
                    value={form.targetWords}
                    onChange={(e) =>
                      setForm((f) => ({ ...f, targetWords: Number(e.target.value) }))
                    }
                    className="w-full rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none"
                  >
                    {targetWordOptions.map((o) => (
                      <option key={o.value} value={o.value}>
                        {o.label}
                      </option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="mb-1 block text-xs text-studio-muted">每章字数</label>
                  <select
                    value={form.chapterWords}
                    onChange={(e) =>
                      setForm((f) => ({ ...f, chapterWords: Number(e.target.value) }))
                    }
                    className="w-full rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none"
                  >
                    {chapterWordOptions.map((o) => (
                      <option key={o.value} value={o.value}>
                        {o.label}
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              <div>
                <label className="mb-1 block text-xs text-studio-muted">故事简介（选填）</label>
                <textarea
                  value={form.synopsis}
                  onChange={(e) => setForm((f) => ({ ...f, synopsis: e.target.value }))}
                  placeholder="一句话或一小段梗概，会写入总纲"
                  rows={3}
                  className="w-full resize-none rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none focus:border-studio-accent"
                />
              </div>

              <div>
                <label className="mb-1 block text-xs text-studio-muted">保存目录</label>
                <div className="flex gap-2">
                  <input
                    value={form.dir}
                    onChange={(e) => setForm((f) => ({ ...f, dir: e.target.value }))}
                    placeholder="/path/to/my-novel"
                    className="flex-1 rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none focus:border-studio-accent"
                  />
                  <button
                    type="button"
                    onClick={async () => {
                      const p = await app().PickCreateDirectory();
                      if (p) setForm((f) => ({ ...f, dir: p }));
                    }}
                    className="shrink-0 rounded-lg border border-studio-border px-3 text-sm"
                  >
                    选择
                  </button>
                </div>
              </div>
            </div>

            <div className="mt-6 flex justify-end gap-3">
              <button
                type="button"
                onClick={closeCreate}
                className="rounded-lg px-4 py-2 text-sm text-studio-muted"
              >
                取消
              </button>
              <button
                type="button"
                onClick={handleCreate}
                disabled={!canCreate || creating}
                className="rounded-lg bg-studio-accent px-4 py-2 text-sm font-medium text-studio-on-accent disabled:opacity-40"
              >
                {creating ? "创建中…" : "创建"}
              </button>
            </div>
              </>
            )}
          </div>
        </div>
      )}

      <SettingsDialog open={settingsOpen} onClose={() => setSettingsOpen(false)} />
    </div>
  );
}
