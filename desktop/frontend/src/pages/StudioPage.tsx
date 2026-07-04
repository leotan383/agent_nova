import { useCallback, useEffect, useRef, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import {
  ArrowLeft,
  BookMarked,
  Brain,
  ChevronDown,
  Download,
  FileText,
  GitBranch,
  LayoutDashboard,
  Map,
  Pencil,
  Plus,
  ScrollText,
  Search,
  Settings,
  Trash2,
} from "lucide-react";
import { ChapterDTO, NovelCard, SearchHitDTO, SettingCategoryDTO, StatusReport, WikiEntryDTO, app, phaseLabel } from "../lib/wails";
import {
  META_SYNOPSIS_ID,
  SettingCategory,
  buildSubdirToCategory,
  classifySettingEntry,
  countByCategory,
  listSidebarOutlineEntries,
} from "../lib/wikiCategories";
import { settingCategoryIcon } from "../lib/settingCategoryIcons";
import CreateCategoryDialog from "../components/CreateCategoryDialog";
import CreateSettingDialog from "../components/CreateSettingDialog";
import ConfirmDialog from "../components/ConfirmDialog";
import ChaptersPanel, { ChapterDocTab, ChaptersView } from "../components/ChaptersPanel";
import ForeshadowPanel, { ForeshadowFocus } from "../components/ForeshadowPanel";
import MemoryPanel, { MemoryFocus } from "../components/MemoryPanel";
import OverviewPanel from "../components/OverviewPanel";
import VolumePlanPanel from "../components/VolumePlanPanel";
import ExportDialog from "../components/ExportDialog";
import SettingsDialog from "../components/SettingsDialog";
import SearchDialog, { SearchSession } from "../components/SearchDialog";
import ThemeToggle from "../components/ThemeToggle";
import WikiPanel from "../components/WikiPanel";
import UnsavedChangesDialog from "../components/UnsavedChangesDialog";
import { confirmUnsavedLeave, hasUnsavedChanges } from "../lib/unsavedGuard";

type Tab = "overview" | "planning" | "chapters" | "memory" | "foreshadow" | "wiki";

type NavSnapshot = {
  tab: Tab;
  chaptersView: ChaptersView;
  selectedChapter: number | null;
  chapterDocTab: ChapterDocTab;
  memoryFocus: MemoryFocus;
  foreshadowFocus: ForeshadowFocus;
  wikiSelectedID: string;
  wikiCategory: SettingCategory | null;
  wikiThemeOnly: boolean;
};

const defaultSettingCategories: SettingCategoryDTO[] = [
  { id: "角色", label: "角色", subdir: "角色", builtin: true },
  { id: "世界观", label: "世界观", subdir: "世界", builtin: true },
  { id: "势力", label: "势力", subdir: "势力", builtin: true },
  { id: "地点", label: "地点", subdir: "地点", builtin: true },
  { id: "物品", label: "物品", subdir: "物品", builtin: true },
];

export default function StudioPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const pendingOpenWrite = useRef(false);
  const [novels, setNovels] = useState<NovelCard[]>([]);
  const [activeId, setActiveId] = useState("");
  const [status, setStatus] = useState<StatusReport | null>(null);
  const [chapters, setChapters] = useState<ChapterDTO[]>([]);
  const [selectedChapter, setSelectedChapter] = useState<number | null>(null);
  const [chapterDocTab, setChapterDocTab] = useState<ChapterDocTab>("body");
  const [chaptersView, setChaptersView] = useState<ChaptersView>("read");
  const [tab, setTab] = useState<Tab>("overview");
  const [memoryFocus, setMemoryFocus] = useState<MemoryFocus>("memories");
  const [foreshadowFocus, setForeshadowFocus] = useState<ForeshadowFocus>("open");
  const [switcherOpen, setSwitcherOpen] = useState(false);
  const [error, setError] = useState("");
  const [versionPanelOpen, setVersionPanelOpen] = useState(false);
  const [searchOpen, setSearchOpen] = useState(false);
  const [searchSession, setSearchSession] = useState<SearchSession | null>(null);
  const [navBeforeSearch, setNavBeforeSearch] = useState<NavSnapshot | null>(null);
  const [searchHighlightId, setSearchHighlightId] = useState("");
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [exportOpen, setExportOpen] = useState(false);
  const [wikiSelectedID, setWikiSelectedID] = useState("");
  const [wikiCategory, setWikiCategory] = useState<SettingCategory | null>(null);
  const [wikiThemeOnly, setWikiThemeOnly] = useState(false);
  const [settingCategories, setSettingCategories] = useState<SettingCategoryDTO[]>(defaultSettingCategories);
  const [wikiCounts, setWikiCounts] = useState<Record<string, number>>({});
  const [wikiEntries, setWikiEntries] = useState<WikiEntryDTO[]>([]);
  const [chapterRefreshKey, setChapterRefreshKey] = useState(0);
  const [healthRefreshKey, setHealthRefreshKey] = useState(0);
  const [planFocusVolume, setPlanFocusVolume] = useState<number | null>(null);
  const [autoReviewChapter, setAutoReviewChapter] = useState<number | null>(null);
  const [createSettingOpen, setCreateSettingOpen] = useState(false);
  const [createSettingCategory, setCreateSettingCategory] = useState<SettingCategory>("角色");
  const [createCategoryOpen, setCreateCategoryOpen] = useState(false);
  const [renameCategoryId, setRenameCategoryId] = useState("");
  const [renameCategoryOpen, setRenameCategoryOpen] = useState(false);
  const [deleteCategoryTarget, setDeleteCategoryTarget] = useState<SettingCategoryDTO | null>(null);

  const loadStudio = useCallback(async () => {
    setError("");
    try {
      const [list, active, st, chs] = await Promise.all([
        app().ListNovels(false),
        app().GetActiveNovel(),
        app().GetStatus(),
        app().ListChapters(),
      ]);
      setNovels(list);
      setActiveId(active.id || "");
      setStatus(st);
      setChapters(chs);
      if (!active.id) {
        navigate("/");
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, [navigate]);

  useEffect(() => {
    loadStudio();
  }, [loadStudio]);

  const loadWikiMeta = useCallback(async () => {
    try {
      const [entries, categories] = await Promise.all([
        app().ListWikiEntries(),
        app().ListSettingCategories(),
      ]);
      setWikiEntries(entries);
      setSettingCategories(categories.length > 0 ? categories : defaultSettingCategories);
      setWikiCounts(countByCategory(entries, categories.length > 0 ? categories : defaultSettingCategories));
    } catch {
      /* 侧边栏计数非关键路径 */
    }
  }, []);

  useEffect(() => {
    if (activeId) void loadWikiMeta();
  }, [activeId, healthRefreshKey, loadWikiMeta]);

  const activeNovel = novels.find((n) => n.id === activeId);

  const resolveWikiNav = (wikiID: string, entries: WikiEntryDTO[], categories: SettingCategoryDTO[]) => {
    const subdirMap = buildSubdirToCategory(categories);
    const themeOnly = wikiID === META_SYNOPSIS_ID || wikiID.startsWith("outline:");
    if (themeOnly) {
      return { wikiCategory: null as SettingCategory | null, wikiThemeOnly: true };
    }
    const entry = entries.find((e) => e.id === wikiID);
    return {
      wikiCategory: entry ? classifySettingEntry(entry, subdirMap) : null,
      wikiThemeOnly: false,
    };
  };

  const switchNovel = async (id: string) => {
    if (id === activeId) {
      setSwitcherOpen(false);
      return;
    }
    const ok = await confirmUnsavedLeave();
    if (!ok) return;
    try {
      await app().SwitchNovel(id);
      setSwitcherOpen(false);
      setSelectedChapter(null);
      setChaptersView("read");
      await loadStudio();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      await loadStudio();
    }
  };

  const loadChapter = (num: number, docKind: ChapterDocTab = "body") => {
    setChaptersView("read");
    setSelectedChapter(num);
    setChapterDocTab(docKind);
    setTab("chapters");
  };

  const guardedLoadChapter = async (num: number, docKind: ChapterDocTab = "body") => {
    if (tab === "chapters" && chaptersView === "read" && selectedChapter === num && chapterDocTab === docKind) {
      return;
    }
    const ok = await confirmUnsavedLeave();
    if (!ok) return;
    setAutoReviewChapter((prev) => (prev === num && docKind === "review" ? prev : null));
    loadChapter(num, docKind);
  };

  const openWriteMode = async () => {
    if (tab === "chapters" && chaptersView === "write") return;
    const ok = await confirmUnsavedLeave();
    if (!ok) return;
    clearSearchReturn();
    setChaptersView("write");
    setTab("chapters");
  };

  useEffect(() => {
    if ((location.state as { openWrite?: boolean } | null)?.openWrite) {
      pendingOpenWrite.current = true;
      navigate("/studio", { replace: true, state: {} });
    }
  }, [location.state, navigate]);

  useEffect(() => {
    if (!pendingOpenWrite.current || !status) return;
    pendingOpenWrite.current = false;
    void openWriteMode();
  }, [status, tab, chaptersView]);

  const refreshChapterView = () => {
    setChapterRefreshKey((k) => k + 1);
    loadStudio();
  };

  const goToChapters = async () => {
    if (tab === "chapters") return;
    const ok = await confirmUnsavedLeave();
    if (!ok) return;
    setTab("chapters");
  };

  const goToCurrentChapter = async () => {
    const num = status?.current_chapter ?? 0;
    if (num <= 0) {
      const latest = chapters[chapters.length - 1]?.number;
      if (latest) {
        await guardedLoadChapter(latest);
      } else {
        await openWriteMode();
      }
      return;
    }
    await guardedLoadChapter(num);
  };

  const goToMemories = async () => {
    if (tab === "memory" && memoryFocus === "memories") return;
    const ok = await confirmUnsavedLeave();
    if (!ok) return;
    setMemoryFocus("memories");
    setTab("memory");
  };

  const goToForeshadows = async () => {
    if (tab === "foreshadow" && foreshadowFocus === "open") return;
    const ok = await confirmUnsavedLeave();
    if (!ok) return;
    setForeshadowFocus("open");
    setTab("foreshadow");
  };

  const goToMemoryConflicts = async () => {
    if (tab === "memory" && memoryFocus === "conflicts") return;
    const ok = await confirmUnsavedLeave();
    if (!ok) return;
    setMemoryFocus("conflicts");
    setTab("memory");
  };

  const goToPlanning = async (volume?: number) => {
    const ok = await confirmUnsavedLeave();
    if (!ok) return;
    if (volume && volume > 0) setPlanFocusVolume(volume);
    clearSearchReturn();
    setTab("planning");
  };

  const handleRebuildIndex = async () => {
    await app().RebuildProjectIndex();
    await loadStudio();
    setHealthRefreshKey((k) => k + 1);
  };

  const handlePlanComplete = () => {
    setHealthRefreshKey((k) => k + 1);
    loadStudio();
  };

  const requestReviewChapter = async (num: number) => {
    const ok = await confirmUnsavedLeave();
    if (!ok) return;
    setAutoReviewChapter(num);
    loadChapter(num, "review");
  };

  const handleReviewComplete = () => {
    setAutoReviewChapter(null);
    setHealthRefreshKey((k) => k + 1);
    refreshChapterView();
  };

  const goToLibrary = async () => {
    const ok = await confirmUnsavedLeave();
    if (!ok) return;
    setSwitcherOpen(false);
    navigate("/");
  };

  const clearSearchReturn = () => {
    setNavBeforeSearch(null);
    setSearchHighlightId("");
  };

  const switchTab = async (id: Tab) => {
    if (tab === id) return;
    const ok = await confirmUnsavedLeave();
    if (!ok) return;
    clearSearchReturn();
    setTab(id);
  };

  const restoreNavBeforeSearch = async () => {
    if (!navBeforeSearch) return;
    const ok = await confirmUnsavedLeave();
    if (!ok) return;
    setTab(navBeforeSearch.tab);
    setChaptersView(navBeforeSearch.chaptersView);
    setSelectedChapter(navBeforeSearch.selectedChapter);
    setChapterDocTab(navBeforeSearch.chapterDocTab);
    setMemoryFocus(navBeforeSearch.memoryFocus);
    setForeshadowFocus(navBeforeSearch.foreshadowFocus);
    setWikiSelectedID(navBeforeSearch.wikiSelectedID);
    setWikiCategory(navBeforeSearch.wikiCategory);
    setWikiThemeOnly(navBeforeSearch.wikiThemeOnly);
    clearSearchReturn();
  };

  const navigateFromSearch = async (hit: SearchHitDTO) => {
    const ok = await confirmUnsavedLeave();
    if (!ok) return;

    setNavBeforeSearch({
      tab,
      chaptersView,
      selectedChapter,
      chapterDocTab,
      memoryFocus,
      foreshadowFocus,
      wikiSelectedID,
      wikiCategory,
      wikiThemeOnly,
    });
    setSearchHighlightId(hit.id);

    switch (hit.kind) {
      case "chapter":
        loadChapter(hit.chapter || parseInt(hit.id, 10) || 1, "body");
        break;
      case "setting":
      case "entity":
        if (hit.wiki_id) {
          const nav = resolveWikiNav(hit.wiki_id, wikiEntries, settingCategories);
          setWikiSelectedID(hit.wiki_id);
          setWikiCategory(nav.wikiCategory);
          setWikiThemeOnly(nav.wikiThemeOnly);
          setTab("wiki");
        }
        break;
      case "memory":
        setMemoryFocus("memories");
        setTab("memory");
        break;
      case "foreshadow":
        setForeshadowFocus("open");
        setTab("foreshadow");
        break;
      default:
        if (hit.chapter > 0) loadChapter(hit.chapter, "body");
    }
  };

  useEffect(() => {
    const onBeforeUnload = (e: BeforeUnloadEvent) => {
      if (!hasUnsavedChanges()) return;
      e.preventDefault();
      e.returnValue = "";
    };
    window.addEventListener("beforeunload", onBeforeUnload);
    return () => window.removeEventListener("beforeunload", onBeforeUnload);
  }, []);

  const goToWikiCategory = async (cat: SettingCategory) => {
    if (tab === "wiki" && wikiCategory === cat && !wikiThemeOnly && !wikiSelectedID) return;
    const ok = await confirmUnsavedLeave();
    if (!ok) return;
    clearSearchReturn();
    setWikiCategory(cat);
    setWikiThemeOnly(false);
    setWikiSelectedID("");
    setTab("wiki");
  };

  const goToWikiEntry = async (wikiID: string) => {
    const ok = await confirmUnsavedLeave();
    if (!ok) return;
    clearSearchReturn();
    const nav = resolveWikiNav(wikiID, wikiEntries, settingCategories);
    setWikiSelectedID(wikiID);
    setWikiCategory(nav.wikiCategory);
    setWikiThemeOnly(nav.wikiThemeOnly);
    setTab("wiki");
  };

  type NavItem = { id: Tab; label: string; icon: typeof LayoutDashboard; hint?: string };

  const openCreateSettingFor = (cat: SettingCategory) => {
    setCreateSettingCategory(cat);
    setCreateSettingOpen(true);
  };

  const handleSettingCreated = async (id: string) => {
    setCreateSettingOpen(false);
    await loadWikiMeta();
    await goToWikiEntry(id);
  };

  const handleCategoryCreated = async (categoryId: string) => {
    setCreateCategoryOpen(false);
    await loadWikiMeta();
    await goToWikiCategory(categoryId);
  };

  const handleCategoryRenamed = async (categoryId: string) => {
    const oldId = renameCategoryId;
    setRenameCategoryOpen(false);
    setRenameCategoryId("");
    await loadWikiMeta();
    if (wikiCategory === oldId) {
      setWikiCategory(categoryId);
    }
    await goToWikiCategory(categoryId);
  };

  const confirmDeleteCategory = async () => {
    if (!deleteCategoryTarget) return;
    const targetId = deleteCategoryTarget.id;
    try {
      await app().DeleteSettingCategory(targetId);
      setDeleteCategoryTarget(null);
      if (wikiCategory === targetId) {
        setWikiCategory(null);
        setWikiSelectedID("");
        if (tab === "wiki") setTab("overview");
      }
      await loadWikiMeta();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const outlineEntries = listSidebarOutlineEntries(wikiEntries);
  const metaNav: NavItem[] = [
    {
      id: "memory",
      label: "记忆",
      icon: Brain,
      hint: "写章时沉淀的可复用知识（角色、世界观、写法等），供后续章节注入上下文；可手动编辑或检测同主题冲突。",
    },
    {
      id: "foreshadow",
      label: "伏笔",
      icon: GitBranch,
      hint: "追踪连载悬念：Open 为尚未在正文兑现的线索，已回收为已完成收束的伏笔，避免长篇连载遗忘。",
    },
  ];

  const renderNavButton = (item: NavItem, indent = false) => {
    const Icon = item.icon;
    const active = tab === item.id;
    return (
      <button
        key={item.id}
        type="button"
        onClick={() => void switchTab(item.id)}
        className={`group relative flex w-full items-center gap-2 rounded-lg py-2 text-sm transition ${
          indent ? "px-3 pl-7" : "px-3"
        } ${
          active
            ? "bg-studio-accent/15 text-studio-accent"
            : "text-studio-muted hover:bg-studio-panel hover:text-studio-text"
        }`}
      >
        <Icon className="h-4 w-4 shrink-0" />
        <span className="min-w-0 flex-1 text-left">{item.label}</span>
        {item.hint && (
          <span
            role="tooltip"
            className="pointer-events-none absolute inset-x-0 top-full z-50 mt-0.5 hidden rounded-md border border-studio-border bg-studio-panel px-2.5 py-2 text-left text-[11px] font-normal leading-snug text-studio-muted shadow-sm group-hover:block"
          >
            {item.hint}
          </span>
        )}
      </button>
    );
  };

  const renderWikiNavButton = (
    wikiID: string,
    label: string,
    icon: typeof ScrollText,
    hint?: string,
  ) => {
    const Icon = icon;
    const active = tab === "wiki" && wikiThemeOnly && wikiSelectedID === wikiID;
    return (
      <button
        key={wikiID}
        type="button"
        onClick={() => goToWikiEntry(wikiID)}
        className={`flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm transition ${
          active
            ? "bg-studio-accent/15 text-studio-accent"
            : "text-studio-muted hover:bg-studio-panel hover:text-studio-text"
        }`}
        title={hint}
      >
        <Icon className="h-4 w-4 shrink-0" />
        <span className="min-w-0 flex-1 truncate text-left">{label}</span>
      </button>
    );
  };

  const renderCreationSection = () => (
    <div>
      <p className="mb-1.5 px-3 text-[10px] font-semibold uppercase tracking-wide text-studio-muted">
        创作
      </p>
      <nav className="space-y-0.5">
        {renderNavButton({
          id: "overview",
          label: "概览",
          icon: LayoutDashboard,
          hint: "作品信息与进度",
        })}
        {renderNavButton({
          id: "planning",
          label: "大纲",
          icon: Map,
          hint: "查看与编辑分卷章纲，已写章节后可 Replan",
        })}
        {outlineEntries.map((e) =>
          renderWikiNavButton(
            e.id,
            e.title,
            ScrollText,
            e.path ? `大纲/${e.title}.md` : undefined,
          ),
        )}
        {renderNavButton({
          id: "chapters",
          label: "正文",
          icon: FileText,
          hint: "章节正文 / 写章",
        })}
      </nav>
    </div>
  );

  const renderStateSection = () => (
    <div>
      <p className="mb-1.5 px-3 text-[10px] font-semibold uppercase tracking-wide text-studio-muted">
        状态
      </p>
      <nav className="space-y-0.5">
        {metaNav.map((item) => renderNavButton(item))}
      </nav>
    </div>
  );

  return (
    <div className="flex h-full min-h-0 flex-col overflow-hidden">
      <header className="flex shrink-0 items-center gap-4 border-b border-studio-border px-5 py-3">
        <button
          type="button"
          onClick={() => void goToLibrary()}
          className="inline-flex items-center gap-1 rounded-lg px-2 py-1.5 text-sm text-studio-muted hover:bg-studio-panel hover:text-studio-text"
        >
          <ArrowLeft className="h-4 w-4" />
          书库
        </button>

        <div className="relative">
          <button
            type="button"
            onClick={() => setSwitcherOpen((v) => !v)}
            className="inline-flex items-center gap-2 rounded-lg border border-studio-border bg-studio-panel px-3 py-2 text-sm hover:border-studio-muted"
          >
            <BookMarked className="h-4 w-4 text-studio-accent" />
            <span className="max-w-[200px] truncate">{activeNovel?.title || status?.title || "未命名"}</span>
            <ChevronDown className="h-4 w-4 text-studio-muted" />
          </button>
          {switcherOpen && (
            <div className="absolute left-0 top-full z-40 mt-2 w-72 rounded-xl border border-studio-border bg-studio-panel py-2 shadow-card">
              {novels.map((n) => (
                <button
                  key={n.id}
                  type="button"
                  onClick={() => switchNovel(n.id)}
                  className={`flex w-full items-center justify-between px-4 py-2.5 text-left text-sm hover:bg-studio-bg ${
                    n.id === activeId ? "text-studio-accent" : ""
                  }`}
                >
                  <span className="truncate">{n.title}</span>
                  <span className="text-xs text-studio-muted">{n.chapter_count}章</span>
                </button>
              ))}
              <div className="my-2 border-t border-studio-border" />
              <button
                type="button"
                onClick={() => void goToLibrary()}
                className="block w-full px-4 py-2 text-left text-sm text-studio-muted hover:text-studio-text"
              >
                返回书库…
              </button>
            </div>
          )}
        </div>

        <span className="rounded-full bg-studio-accent/10 px-2.5 py-0.5 text-xs text-studio-accent">
          {phaseLabel[status?.phase || ""] || status?.phase}
        </span>

        {status && (status.genre || status.style) && (
          <div className="hidden flex-wrap items-center gap-1.5 md:flex">
            {status.genre && (
              <span className="rounded-full bg-studio-panel px-2 py-0.5 text-[11px] text-studio-muted ring-1 ring-studio-border">
                {status.genre}
              </span>
            )}
            {status.style && (
              <span className="rounded-full bg-studio-panel px-2 py-0.5 text-[11px] text-studio-muted ring-1 ring-studio-border">
                {status.style}
              </span>
            )}
            {status.chapter_words_goal > 0 && (
              <span className="rounded-full bg-studio-panel px-2 py-0.5 text-[11px] text-studio-muted ring-1 ring-studio-border">
                约 {status.chapter_words_goal} 字/章
              </span>
            )}
          </div>
        )}

        <div className="ml-auto flex items-center gap-3 text-xs text-studio-muted">
          <button
            type="button"
            onClick={() => setSearchOpen(true)}
            className="inline-flex items-center gap-1.5 rounded-lg border border-studio-border px-3 py-2 text-sm text-studio-muted hover:border-studio-muted hover:text-studio-text"
          >
            <Search className="h-4 w-4" />
            <span className="hidden sm:inline">搜索</span>
          </button>
          <button
            type="button"
            onClick={() => setExportOpen(true)}
            className="inline-flex items-center gap-1.5 rounded-lg border border-studio-border px-3 py-2 text-sm text-studio-muted hover:border-studio-muted hover:text-studio-text"
            title="导出小说"
          >
            <Download className="h-4 w-4" />
            <span className="hidden sm:inline">导出</span>
          </button>
          <button
            type="button"
            onClick={() => setSettingsOpen(true)}
            className="inline-flex items-center gap-1.5 rounded-lg border border-studio-border px-3 py-2 text-sm text-studio-muted hover:border-studio-muted hover:text-studio-text"
            title="应用设置"
          >
            <Settings className="h-4 w-4" />
          </button>
          <ThemeToggle />
          {status && status.target_words > 0 && (
            <span className="hidden tabular-nums sm:inline">
              {status.progress_percent.toFixed(0)}% · {status.chapter_count} 章
            </span>
          )}
          <span>
            {status?.chapter_count ?? 0} 章 · {status?.open_foreshadows ?? 0} 条 open 伏笔
          </span>
        </div>
      </header>

      {error && (
        <div className="mx-5 mt-4 shrink-0 studio-alert-error">
          {error}
        </div>
      )}

      <div className="flex min-h-0 flex-1 overflow-hidden">
        <aside className="w-52 shrink-0 overflow-y-auto border-r border-studio-border p-4">
          <div className="space-y-4">
            {renderCreationSection()}
            <div>
              <div className="mb-1.5 flex items-center justify-between px-3">
                <p className="text-[10px] font-semibold uppercase tracking-wide text-studio-muted">
                  设定
                </p>
                <button
                  type="button"
                  onClick={() => setCreateCategoryOpen(true)}
                  className="rounded-md p-1 text-studio-muted transition hover:bg-studio-panel hover:text-studio-accent"
                  title="新建设定分类"
                >
                  <Plus className="h-3.5 w-3.5" />
                </button>
              </div>

              <nav className="space-y-0.5">
                {settingCategories.map((cat) => {
                  const Icon = settingCategoryIcon(cat.id);
                  const count = wikiCounts[cat.id] ?? 0;
                  const active = tab === "wiki" && wikiCategory === cat.id && !wikiThemeOnly;
                  return (
                    <div key={cat.id} className="group flex items-center gap-0.5">
                      <button
                        type="button"
                        onClick={() => void goToWikiCategory(cat.id)}
                        className={`flex min-w-0 flex-1 items-center gap-2 rounded-lg px-3 py-2 text-sm transition ${
                          active
                            ? "bg-studio-accent/15 text-studio-accent"
                            : "text-studio-muted hover:bg-studio-panel hover:text-studio-text"
                        }`}
                      >
                        <Icon className="h-4 w-4 shrink-0" />
                        <span className="min-w-0 flex-1 truncate text-left">{cat.label}</span>
                        {count > 0 && (
                          <span className="text-xs tabular-nums text-studio-muted/70">({count})</span>
                        )}
                      </button>
                      <button
                        type="button"
                        onClick={() => openCreateSettingFor(cat.id)}
                        className="shrink-0 rounded-md p-1 text-studio-muted opacity-0 transition hover:bg-studio-panel hover:text-studio-accent group-hover:opacity-100"
                        title={`新建${cat.label}设定`}
                      >
                        <Plus className="h-3.5 w-3.5" />
                      </button>
                      {!cat.builtin && (
                        <>
                          <button
                            type="button"
                            onClick={() => {
                              setRenameCategoryId(cat.id);
                              setRenameCategoryOpen(true);
                            }}
                            className="shrink-0 rounded-md p-1 text-studio-muted opacity-0 transition hover:bg-studio-panel hover:text-studio-text group-hover:opacity-100"
                            title={`重命名「${cat.label}」`}
                          >
                            <Pencil className="h-3 w-3" />
                          </button>
                          <button
                            type="button"
                            onClick={() => setDeleteCategoryTarget(cat)}
                            className="shrink-0 rounded-md p-1 text-studio-muted opacity-0 transition hover:bg-[rgb(var(--studio-danger-bg))] hover:text-[rgb(var(--studio-danger-fg))] group-hover:opacity-100"
                            title={`删除「${cat.label}」`}
                          >
                            <Trash2 className="h-3 w-3" />
                          </button>
                        </>
                      )}
                    </div>
                  );
                })}
              </nav>
            </div>
            {renderStateSection()}
          </div>
          <p className="mt-6 px-3 text-[11px] leading-relaxed text-studio-muted/80">
            创作：概览、大纲与正文；设定：角色、世界观等分类；状态：记忆与伏笔。
          </p>
        </aside>

        <main className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden p-6">
          {navBeforeSearch && (
            <div className="mb-4 flex shrink-0 flex-wrap items-center gap-2 rounded-lg border border-studio-border bg-studio-panel px-3 py-2">
              <button
                type="button"
                onClick={restoreNavBeforeSearch}
                className="inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-sm text-studio-accent hover:bg-studio-accent/10"
              >
                <ArrowLeft className="h-4 w-4" />
                返回
              </button>
              {searchSession && (
                <button
                  type="button"
                  onClick={() => setSearchOpen(true)}
                  className="inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-sm text-studio-muted hover:bg-studio-bg hover:text-studio-text"
                >
                  <Search className="h-3.5 w-3.5" />
                  继续搜索「{searchSession.query}」
                </button>
              )}
              <span className="text-xs text-studio-muted">从搜索结果跳转</span>
            </div>
          )}

          {tab === "overview" && status && (
            <div className="min-h-0 flex-1 overflow-y-auto">
              <OverviewPanel
                novelId={activeId}
                status={status}
                healthRefreshKey={healthRefreshKey}
                onContinueWrite={() => void openWriteMode()}
                onOpenPlanning={(vol) => void goToPlanning(vol)}
                onOpenWrite={() => void openWriteMode()}
                onOpenSettings={() => setSettingsOpen(true)}
                onRebuildIndex={handleRebuildIndex}
                onReviewChapter={(num) => void requestReviewChapter(num)}
                onGoToChapters={() => void goToChapters()}
                onGoToCurrentChapter={() => void goToCurrentChapter()}
                onGoToMemories={() => void goToMemories()}
                onGoToForeshadows={() => void goToForeshadows()}
                onGoToMemoryConflicts={() => void goToMemoryConflicts()}
                onProjectToolsRefresh={() => {
                  setHealthRefreshKey((k) => k + 1);
                  loadStudio();
                }}
              />
            </div>
          )}

          {tab === "planning" && status && (
            <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
              <VolumePlanPanel
                suggestedVolume={status.current_volume || 1}
                currentChapter={status.current_chapter}
                focusVolume={planFocusVolume}
                onComplete={handlePlanComplete}
              />
            </div>
          )}

          {tab === "chapters" && (
            <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
              <ChaptersPanel
                status={status}
                chapters={chapters}
                view={chaptersView}
                selectedChapter={selectedChapter}
                chapterDocTab={chapterDocTab}
                chapterRefreshKey={chapterRefreshKey}
                autoReviewChapter={autoReviewChapter}
                versionPanelOpen={versionPanelOpen}
                onVersionPanelOpenChange={setVersionPanelOpen}
                onSelectChapter={(num) => void guardedLoadChapter(num)}
                onStartWrite={() => void openWriteMode()}
                onWriteComplete={() => {
                  loadStudio();
                  setHealthRefreshKey((k) => k + 1);
                }}
                onGoToPlanning={(vol) => void goToPlanning(vol)}
                onReviewChapter={(num) => void requestReviewChapter(num)}
                onReadChapter={(num) => void guardedLoadChapter(num)}
                onChapterSaved={refreshChapterView}
                onReviewComplete={handleReviewComplete}
                onRebuildIndex={handleRebuildIndex}
              />
            </div>
          )}

          {tab === "memory" && (
            <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
              <MemoryPanel
                focus={memoryFocus}
                onFocusChange={setMemoryFocus}
                highlightId={searchHighlightId}
              />
            </div>
          )}

          {tab === "foreshadow" && (
            <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
              <ForeshadowPanel
                focus={foreshadowFocus}
                onFocusChange={setForeshadowFocus}
                highlightId={searchHighlightId}
              />
            </div>
          )}

          {tab === "wiki" && (
            <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
              <WikiPanel
                status={status}
                initialSelectedID={wikiSelectedID}
                settingCategories={settingCategories}
                categoryFilter={wikiThemeOnly ? null : wikiCategory}
                themeOnly={wikiThemeOnly}
              />
            </div>
          )}
        </main>
      </div>

      <CreateSettingDialog
        open={createSettingOpen}
        category={createSettingCategory}
        categories={settingCategories}
        onClose={() => setCreateSettingOpen(false)}
        onCreated={(id) => void handleSettingCreated(id)}
      />
      <CreateCategoryDialog
        open={createCategoryOpen}
        onClose={() => setCreateCategoryOpen(false)}
        onCreated={(id) => void handleCategoryCreated(id)}
      />
      <CreateCategoryDialog
        open={renameCategoryOpen}
        mode="rename"
        categoryId={renameCategoryId}
        onClose={() => {
          setRenameCategoryOpen(false);
          setRenameCategoryId("");
        }}
        onCreated={() => {}}
        onRenamed={(id) => void handleCategoryRenamed(id)}
      />
      <ConfirmDialog
        open={!!deleteCategoryTarget}
        title="删除设定分类"
        message={
          deleteCategoryTarget
            ? `确定删除「${deleteCategoryTarget.label}」？${
                (wikiCounts[deleteCategoryTarget.id] ?? 0) > 0
                  ? ` 该分类下 ${wikiCounts[deleteCategoryTarget.id]} 条设定将一并删除，且不可恢复。`
                  : " 空分类将被移除。"
              }`
            : ""
        }
        confirmLabel="删除"
        destructive
        onConfirm={() => void confirmDeleteCategory()}
        onCancel={() => setDeleteCategoryTarget(null)}
      />
      <SettingsDialog open={settingsOpen} onClose={() => setSettingsOpen(false)} />
      <ExportDialog open={exportOpen} onClose={() => setExportOpen(false)} status={status} />

      <SearchDialog
        open={searchOpen}
        onClose={() => setSearchOpen(false)}
        session={searchSession}
        onSessionChange={setSearchSession}
        onNavigate={navigateFromSearch}
      />
      <UnsavedChangesDialog />
    </div>
  );
}
