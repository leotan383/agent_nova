import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  Archive,
  FolderOpen,
  Plus,
  Search,
  Settings,
  SlidersHorizontal,
  Sparkles,
  X,
} from "lucide-react";
import ConfirmDialog from "../components/ConfirmDialog";
import DiscoverCreatePanel from "../components/DiscoverCreatePanel";
import InspirationLibraryPanel from "../components/InspirationLibraryPanel";
import InspirationPicker from "../components/InspirationPicker";
import NovelCardView from "../components/NovelCard";
import SettingsDialog from "../components/SettingsDialog";
import ThemeToggle from "../components/ThemeToggle";
import {
  LibraryFilters,
  LibrarySortKey,
  filterNovels,
  libraryStats,
  sortNovels,
} from "../lib/libraryUtils";
import { prefillToCreateForm } from "../lib/inspirationUtils";
import {
  InspirationPrefillDTO,
  NovelCard,
  app,
  chapterWordOptions,
  formatWordCount,
  genreOptions,
  phaseLabel,
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

const sortOptions: { id: LibrarySortKey; label: string }[] = [
  { id: "last_opened", label: "最近打开" },
  { id: "progress", label: "进度" },
  { id: "chapters", label: "章数" },
  { id: "title", label: "书名" },
];

const phaseFilterOptions = [
  { value: "", label: "全部阶段" },
  { value: "init_done", label: phaseLabel.init_done },
  { value: "planning", label: phaseLabel.planning },
  { value: "writing", label: phaseLabel.writing },
  { value: "paused", label: phaseLabel.paused },
];

type ConfirmState = {
  title: string;
  message: string;
  confirmLabel: string;
  destructive?: boolean;
  onConfirm: () => void;
};

export default function LibraryPage() {
  const navigate = useNavigate();
  const [pageTab, setPageTab] = useState<"novels" | "inspirations">("novels");
  const [novels, setNovels] = useState<NovelCard[]>([]);
  const [activeId, setActiveId] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const [createMode, setCreateMode] = useState<"form" | "discover">("discover");
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [form, setForm] = useState(defaultCreateForm);
  const [sort, setSort] = useState<LibrarySortKey>("last_opened");
  const [filters, setFilters] = useState<LibraryFilters>({
    query: "",
    genre: "",
    phase: "",
    showArchived: false,
  });
  const [filterOpen, setFilterOpen] = useState(false);
  const [confirm, setConfirm] = useState<ConfirmState | null>(null);
  const [createInspiration, setCreateInspiration] = useState<InspirationPrefillDTO | null>(null);
  const filterRef = useRef<HTMLDivElement>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [list, active] = await Promise.all([
        app().ListNovels(true),
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

  useEffect(() => {
    if (!filterOpen) return;
    const close = (e: MouseEvent) => {
      if (filterRef.current && !filterRef.current.contains(e.target as Node)) {
        setFilterOpen(false);
      }
    };
    document.addEventListener("mousedown", close);
    return () => document.removeEventListener("mousedown", close);
  }, [filterOpen]);

  const archivedCount = useMemo(() => novels.filter((n) => n.archived).length, [novels]);
  const stats = useMemo(() => libraryStats(novels), [novels]);
  const hasActiveFilters = !!(filters.query || filters.genre || filters.phase);

  const filtered = useMemo(() => {
    return sortNovels(filterNovels(novels, filters), sort);
  }, [novels, filters, sort]);

  const continueNovel = useMemo(() => {
    if (filters.showArchived || hasActiveFilters) return null;
    const active = novels.find((n) => n.id === activeId && !n.archived && !n.missing);
    if (active) return active;
    return (
      sortNovels(
        novels.filter((n) => !n.archived && !n.missing),
        "last_opened",
      )[0] ?? null
    );
  }, [novels, activeId, filters.showArchived, hasActiveFilters]);

  const filterCount = (filters.genre ? 1 : 0) + (filters.phase ? 1 : 0) + (sort !== "last_opened" ? 1 : 0);

  const gridNovels = useMemo(() => {
    if (filters.showArchived || hasActiveFilters || !continueNovel) return filtered;
    if (!filtered.some((n) => n.id === continueNovel.id)) return filtered;
    return [continueNovel, ...filtered.filter((n) => n.id !== continueNovel.id)];
  }, [filtered, continueNovel, filters.showArchived, hasActiveFilters]);

  const openNovel = async (id: string) => {
    try {
      await app().SwitchNovel(id);
      navigate("/studio");
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      await refresh();
    }
  };

  const openNovelAndWrite = async (id: string) => {
    try {
      await app().SwitchNovel(id);
      navigate("/studio", { state: { openWrite: true } });
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
    setCreateInspiration(null);
    app().ClearDiscover().catch(() => {});
  };

  const applyInspirationPrefill = (prefill: InspirationPrefillDTO | null) => {
    setCreateInspiration(prefill);
    if (!prefill) return;
    const next = prefillToCreateForm(prefill);
    setForm((f) => ({
      ...f,
      title: next.title,
      genre: next.genre,
      style: next.style,
      synopsis: next.synopsis,
    }));
  };

  const openCreateFromInspiration = async (inspirationId: string) => {
    try {
      const prefill = await app().GetInspirationPrefill(inspirationId);
      applyInspirationPrefill(prefill);
      setCreateMode("form");
      setShowCreate(true);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const handleCreate = async () => {
    if (!form.dir.trim() || !form.title.trim()) {
      setError(!form.dir.trim() ? "请选择保存目录" : "书名不能为空");
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
        inspiration_id: createInspiration?.inspiration_id,
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

  const askRemove = (novel: NovelCard) => {
    setConfirm({
      title: "从书库移除",
      message: `确定移除「${novel.title}」？文件不会被删除。`,
      confirmLabel: "移除",
      destructive: true,
      onConfirm: async () => {
        setConfirm(null);
        await app().RemoveFromLibrary(novel.id);
        await refresh();
      },
    });
  };

  const askArchive = (novel: NovelCard) => {
    const archiving = !novel.archived;
    setConfirm({
      title: archiving ? "归档此书" : "取消归档",
      message: archiving ? `「${novel.title}」将移入归档，不再显示在主列表。` : `「${novel.title}」将恢复到主列表。`,
      confirmLabel: archiving ? "归档" : "恢复",
      onConfirm: async () => {
        setConfirm(null);
        await app().SetNovelArchived(novel.id, archiving);
        await refresh();
      },
    });
  };

  const cardProps = (n: NovelCard) => ({
    novel: n,
    continueActive: !filters.showArchived && !hasActiveFilters && continueNovel?.id === n.id,
    onOpen: () => openNovel(n.id),
    onContinueWrite: () => openNovelAndWrite(n.id),
    onReveal: () => app().RevealInFolder(n.path),
    onRemove: () => askRemove(n),
    onTogglePin: async () => {
      await app().SetNovelPinned(n.id, !n.pinned);
      await refresh();
    },
    onToggleArchive: () => askArchive(n),
  });

  const canCreate = form.dir.trim() !== "" && form.title.trim() !== "";

  return (
    <div className="min-h-full bg-studio-bg">
      <div className="mx-auto max-w-5xl px-6 pb-16 pt-10">
        <div className="mb-6 flex gap-1 rounded-xl border border-studio-border bg-studio-panel p-1">
          <button
            type="button"
            onClick={() => setPageTab("novels")}
            className={`flex-1 rounded-lg px-4 py-2 text-sm font-medium transition ${
              pageTab === "novels" ? "bg-studio-bg text-studio-accent shadow-sm" : "text-studio-muted"
            }`}
          >
            书库
          </button>
          <button
            type="button"
            onClick={() => setPageTab("inspirations")}
            className={`flex-1 rounded-lg px-4 py-2 text-sm font-medium transition ${
              pageTab === "inspirations" ? "bg-studio-bg text-studio-accent shadow-sm" : "text-studio-muted"
            }`}
          >
            灵感库
          </button>
        </div>

        {pageTab === "inspirations" ? (
          <InspirationLibraryPanel
            onCreateNovel={(id) => void openCreateFromInspiration(id)}
            onOpenNovel={(id) => void openNovel(id)}
          />
        ) : (
          <>
        {/* Header */}
        <header className="mb-8 flex items-start justify-between gap-4">
          <div>
            <h1 className="text-2xl font-semibold tracking-tight text-studio-text">书库</h1>
            {!loading && stats.count > 0 && (
              <p className="mt-1 text-sm text-studio-muted">
                {stats.count} 本书
                {stats.totalWords > 0 && ` · ${formatWordCount(stats.totalWords)}`}
              </p>
            )}
          </div>
          <div className="flex items-center gap-1.5">
            <ThemeToggle />
            <button
              type="button"
              onClick={() => setSettingsOpen(true)}
              className="rounded-lg p-2 text-studio-muted hover:bg-studio-panel hover:text-studio-text"
              title="设置"
            >
              <Settings className="h-4 w-4" />
            </button>
            <button
              type="button"
              onClick={handleOpenExisting}
              className="hidden rounded-lg p-2 text-studio-muted hover:bg-studio-panel hover:text-studio-text sm:block"
              title="打开已有项目"
            >
              <FolderOpen className="h-4 w-4" />
            </button>
            <button
              type="button"
              onClick={() => setShowCreate(true)}
              className="ml-1 inline-flex items-center gap-1.5 rounded-xl bg-studio-accent px-4 py-2 text-sm font-medium text-studio-on-accent hover:brightness-110"
            >
              <Plus className="h-4 w-4" />
              新建
            </button>
          </div>
        </header>

        {error && !showCreate && (
          <div className="mb-6 studio-alert-error-compact">{error}</div>
        )}

        {/* Search + filter */}
        {!loading && novels.some((n) => !n.archived || filters.showArchived) && (
          <div className="mb-6 flex gap-2">
            <div className="relative min-w-0 flex-1">
              <Search className="pointer-events-none absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-studio-muted/70" />
              <input
                value={filters.query}
                onChange={(e) => setFilters((f) => ({ ...f, query: e.target.value }))}
                placeholder="搜索书名…"
                className="w-full rounded-xl border-0 bg-studio-panel py-2.5 pl-10 pr-9 text-sm shadow-sm ring-1 ring-studio-border/80 outline-none transition focus:ring-studio-accent/40"
              />
              {filters.query && (
                <button
                  type="button"
                  onClick={() => setFilters((f) => ({ ...f, query: "" }))}
                  className="absolute right-2.5 top-1/2 -translate-y-1/2 rounded p-0.5 text-studio-muted hover:text-studio-text"
                >
                  <X className="h-3.5 w-3.5" />
                </button>
              )}
            </div>
            <div className="relative" ref={filterRef}>
              <button
                type="button"
                onClick={() => setFilterOpen((v) => !v)}
                className={`inline-flex items-center gap-1.5 rounded-xl px-3 py-2.5 text-sm ring-1 transition ${
                  filterOpen || filterCount > 0
                    ? "bg-studio-panel text-studio-text ring-studio-accent/30"
                    : "bg-studio-panel text-studio-muted ring-studio-border/80 hover:text-studio-text"
                }`}
              >
                <SlidersHorizontal className="h-4 w-4" />
                <span className="hidden sm:inline">筛选</span>
                {filterCount > 0 && (
                  <span className="rounded-full bg-studio-accent/15 px-1.5 text-[10px] text-studio-accent">
                    {filterCount}
                  </span>
                )}
              </button>
              {filterOpen && (
                <div className="absolute right-0 top-full z-30 mt-2 w-56 rounded-xl border border-studio-border bg-studio-panel p-3 shadow-card">
                  <label className="mb-1 block text-[11px] text-studio-muted">题材</label>
                  <select
                    value={filters.genre}
                    onChange={(e) => setFilters((f) => ({ ...f, genre: e.target.value }))}
                    className="mb-3 w-full rounded-lg border border-studio-border bg-studio-bg px-2.5 py-1.5 text-sm outline-none"
                  >
                    <option value="">全部</option>
                    {genreOptions.map((g) => (
                      <option key={g} value={g}>
                        {g}
                      </option>
                    ))}
                  </select>
                  <label className="mb-1 block text-[11px] text-studio-muted">阶段</label>
                  <select
                    value={filters.phase}
                    onChange={(e) => setFilters((f) => ({ ...f, phase: e.target.value }))}
                    className="mb-3 w-full rounded-lg border border-studio-border bg-studio-bg px-2.5 py-1.5 text-sm outline-none"
                  >
                    {phaseFilterOptions.map((o) => (
                      <option key={o.value || "all"} value={o.value}>
                        {o.label}
                      </option>
                    ))}
                  </select>
                  <label className="mb-1 block text-[11px] text-studio-muted">排序</label>
                  <select
                    value={sort}
                    onChange={(e) => setSort(e.target.value as LibrarySortKey)}
                    className="w-full rounded-lg border border-studio-border bg-studio-bg px-2.5 py-1.5 text-sm outline-none"
                  >
                    {sortOptions.map((o) => (
                      <option key={o.id} value={o.id}>
                        {o.label}
                      </option>
                    ))}
                  </select>
                  {hasActiveFilters && (
                    <button
                      type="button"
                      onClick={() => {
                        setFilters((f) => ({ ...f, genre: "", phase: "" }));
                        setSort("last_opened");
                      }}
                      className="mt-3 w-full text-center text-xs text-studio-accent hover:underline"
                    >
                      重置筛选
                    </button>
                  )}
                </div>
              )}
            </div>
          </div>
        )}

        {/* Content */}
        {loading ? (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {[1, 2, 3].map((i) => (
              <div key={i} className="h-52 animate-pulse rounded-2xl bg-studio-panel ring-1 ring-studio-border/60" />
            ))}
          </div>
        ) : filters.showArchived ? (
          filtered.length === 0 ? (
            <p className="py-20 text-center text-sm text-studio-muted">没有已归档的书</p>
          ) : (
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {filtered.map((n) => (
                <NovelCardView key={n.id} {...cardProps(n)} />
              ))}
            </div>
          )
        ) : novels.filter((n) => !n.archived).length === 0 ? (
          <div className="flex flex-col items-center py-20 text-center">
            <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-studio-accent/10">
              <Sparkles className="h-7 w-7 text-studio-accent" />
            </div>
            <h2 className="text-lg font-medium">开始第一个故事</h2>
            <p className="mt-2 max-w-md text-sm leading-relaxed text-studio-muted">
              三步完成第一本小说：探讨立项 → 规划卷纲 → AI 写第一章。
            </p>
            <ol className="mt-6 max-w-sm space-y-2 text-left text-sm text-studio-muted">
              <li className="flex gap-2">
                <span className="font-medium text-studio-accent">1.</span>
                <span>点击「新建小说」，用 AI 探讨或表单创建设定</span>
              </li>
              <li className="flex gap-2">
                <span className="font-medium text-studio-accent">2.</span>
                <span>进入工作室 → 规划 Tab，生成第 1 卷卷纲</span>
              </li>
              <li className="flex gap-2">
                <span className="font-medium text-studio-accent">3.</span>
                <span>章节 Tab → AI 写章，约 30 分钟产出第一章</span>
              </li>
            </ol>
            <div className="mt-8 flex gap-3">
              <button
                type="button"
                onClick={() => setShowCreate(true)}
                className="rounded-xl bg-studio-accent px-5 py-2 text-sm font-medium text-studio-on-accent"
              >
                新建小说
              </button>
              <button
                type="button"
                onClick={handleOpenExisting}
                className="rounded-xl px-5 py-2 text-sm text-studio-muted ring-1 ring-studio-border hover:text-studio-text"
              >
                打开已有
              </button>
            </div>
          </div>
        ) : filtered.length === 0 ? (
          <div className="py-20 text-center">
            <p className="text-sm text-studio-muted">没有匹配的书</p>
            <button
              type="button"
              onClick={() => setFilters((f) => ({ ...f, query: "", genre: "", phase: "" }))}
              className="mt-2 text-sm text-studio-accent hover:underline"
            >
              清除搜索
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {gridNovels.map((n) => (
              <NovelCardView key={n.id} {...cardProps(n)} />
            ))}
          </div>
        )}

        {/* Footer links */}
        {!loading && archivedCount > 0 && (
          <div className="mt-10 text-center">
            <button
              type="button"
              onClick={() =>
                setFilters((f) => ({
                  ...f,
                  showArchived: !f.showArchived,
                  query: "",
                  genre: "",
                  phase: "",
                }))
              }
              className="inline-flex items-center gap-1.5 text-xs text-studio-muted transition hover:text-studio-text"
            >
              <Archive className="h-3.5 w-3.5" />
              {filters.showArchived ? "返回书库" : `已归档 · ${archivedCount}`}
            </button>
          </div>
        )}
          </>
        )}
      </div>

      {/* Create modal */}
      {showCreate && (
        <div className="studio-modal-overlay fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-2xl border border-studio-border bg-studio-panel p-6 shadow-card">
            <h2 className="text-lg font-medium">新建小说</h2>
            <div className="mt-4">
              <InspirationPicker
                selectedId={createInspiration?.inspiration_id || ""}
                onSelect={applyInspirationPrefill}
              />
            </div>
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
                  inspirationId={createInspiration?.inspiration_id}
                  seedPrompt={createInspiration?.seed_prompt}
                  initialGenre={createInspiration?.genre || form.genre}
                />
              </div>
            ) : (
              <>
                {error && <div className="mt-4 studio-alert-error-compact">{error}</div>}
                <div className="mt-5 space-y-4">
                  <div>
                    <label className="mb-1 block text-xs text-studio-muted">书名 *</label>
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
                        className="w-full rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm"
                      >
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
                        value={form.style}
                        onChange={(e) => setForm((f) => ({ ...f, style: e.target.value }))}
                        className="w-full rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm"
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
                        onChange={(e) => setForm((f) => ({ ...f, targetWords: Number(e.target.value) }))}
                        className="w-full rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm"
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
                        onChange={(e) => setForm((f) => ({ ...f, chapterWords: Number(e.target.value) }))}
                        className="w-full rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm"
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
                    <label className="mb-1 block text-xs text-studio-muted">简介（选填）</label>
                    <textarea
                      value={form.synopsis}
                      onChange={(e) => setForm((f) => ({ ...f, synopsis: e.target.value }))}
                      rows={2}
                      className="w-full resize-none rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none focus:border-studio-accent"
                    />
                  </div>
                  <div>
                    <label className="mb-1 block text-xs text-studio-muted">保存目录</label>
                    <div className="flex gap-2">
                      <input
                        value={form.dir}
                        onChange={(e) => setForm((f) => ({ ...f, dir: e.target.value }))}
                        className="min-w-0 flex-1 rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm"
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
                  <button type="button" onClick={closeCreate} className="px-4 py-2 text-sm text-studio-muted">
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

      <ConfirmDialog
        open={confirm != null}
        title={confirm?.title ?? ""}
        message={confirm?.message ?? ""}
        confirmLabel={confirm?.confirmLabel}
        destructive={confirm?.destructive}
        onConfirm={() => confirm?.onConfirm()}
        onCancel={() => setConfirm(null)}
      />
      <SettingsDialog open={settingsOpen} onClose={() => setSettingsOpen(false)} />
    </div>
  );
}
