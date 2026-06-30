import { useCallback, useEffect, useState } from "react";
import { Link, useBlocker, useNavigate } from "react-router-dom";
import {
  ArrowLeft,
  BookMarked,
  BookOpen,
  Brain,
  ChevronDown,
  Download,
  FileText,
  History,
  LayoutDashboard,
  PenLine,
  Search,
  Settings,
  Users,
} from "lucide-react";
import { ChapterDTO, NovelCard, SearchHitDTO, StatusReport, app, phaseLabel } from "../lib/wails";
import ChapterDocumentPanel from "../components/ChapterDocumentPanel";
import MemoryPanel, { MemoryFocus } from "../components/MemoryPanel";
import WritePanel from "../components/WritePanel";
import ChapterCoachPanel from "../components/ChapterCoachPanel";
import ChapterVersionPanel from "../components/ChapterVersionPanel";
import ProgressPanel from "../components/ProgressPanel";
import ProjectHealthPanel from "../components/ProjectHealthPanel";
import VolumePlanPanel from "../components/VolumePlanPanel";
import EntityPanel from "../components/EntityPanel";
import ExportDialog from "../components/ExportDialog";
import SettingsDialog from "../components/SettingsDialog";
import SearchDialog, { SearchSession } from "../components/SearchDialog";
import ThemeToggle from "../components/ThemeToggle";
import WikiPanel from "../components/WikiPanel";
import UnsavedChangesDialog from "../components/UnsavedChangesDialog";
import { confirmUnsavedLeave, hasUnsavedChanges } from "../lib/unsavedGuard";

type Tab = "overview" | "write" | "chapters" | "memory" | "wiki" | "entities";
type ChapterDocTab = "body" | "review" | "summary";

type NavSnapshot = {
  tab: Tab;
  selectedChapter: number | null;
  chapterDocTab: ChapterDocTab;
  memoryFocus: MemoryFocus;
  wikiSelectedID: string;
};

export default function StudioPage() {
  const navigate = useNavigate();
  const [novels, setNovels] = useState<NovelCard[]>([]);
  const [activeId, setActiveId] = useState("");
  const [status, setStatus] = useState<StatusReport | null>(null);
  const [chapters, setChapters] = useState<ChapterDTO[]>([]);
  const [selectedChapter, setSelectedChapter] = useState<number | null>(null);
  const [chapterDocTab, setChapterDocTab] = useState<ChapterDocTab>("body");
  const [tab, setTab] = useState<Tab>("overview");
  const [memoryFocus, setMemoryFocus] = useState<MemoryFocus>("memories");
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
  const [chapterRefreshKey, setChapterRefreshKey] = useState(0);
  const [healthRefreshKey, setHealthRefreshKey] = useState(0);
  const [planFocusVolume, setPlanFocusVolume] = useState<number | null>(null);

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

  const activeNovel = novels.find((n) => n.id === activeId);

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
      await loadStudio();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      await loadStudio();
    }
  };

  const loadChapter = (num: number, docKind: ChapterDocTab = "body") => {
    setSelectedChapter(num);
    setChapterDocTab(docKind);
    setTab("chapters");
  };

  const guardedLoadChapter = async (num: number, docKind: ChapterDocTab = "body") => {
    if (tab === "chapters" && selectedChapter === num && chapterDocTab === docKind) return;
    const ok = await confirmUnsavedLeave();
    if (!ok) return;
    loadChapter(num, docKind);
  };

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
        await goToChapters();
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
    if (tab === "memory" && memoryFocus === "foreshadows") return;
    const ok = await confirmUnsavedLeave();
    if (!ok) return;
    setMemoryFocus("foreshadows");
    setTab("memory");
  };

  const goToWrite = async () => {
    if (tab === "write") return;
    const ok = await confirmUnsavedLeave();
    if (!ok) return;
    clearSearchReturn();
    setTab("write");
  };

  const focusPlanVolume = (volume: number) => {
    setPlanFocusVolume(volume);
    setHealthRefreshKey((k) => k + 1);
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
    setSelectedChapter(navBeforeSearch.selectedChapter);
    setChapterDocTab(navBeforeSearch.chapterDocTab);
    setMemoryFocus(navBeforeSearch.memoryFocus);
    setWikiSelectedID(navBeforeSearch.wikiSelectedID);
    clearSearchReturn();
  };

  const navigateFromSearch = async (hit: SearchHitDTO) => {
    const ok = await confirmUnsavedLeave();
    if (!ok) return;

    setNavBeforeSearch({
      tab,
      selectedChapter,
      chapterDocTab,
      memoryFocus,
      wikiSelectedID,
    });
    setSearchHighlightId(hit.id);

    switch (hit.kind) {
      case "chapter":
        loadChapter(hit.chapter || parseInt(hit.id, 10) || 1, "body");
        break;
      case "setting":
      case "entity":
        if (hit.wiki_id) {
          setWikiSelectedID(hit.wiki_id);
          setTab("wiki");
        }
        break;
      case "memory":
        setMemoryFocus("memories");
        setTab("memory");
        break;
      case "foreshadow":
        setMemoryFocus("foreshadows");
        setTab("memory");
        break;
      default:
        if (hit.chapter > 0) loadChapter(hit.chapter, "body");
    }
  };

  const blocker = useBlocker(
    ({ currentLocation, nextLocation }) =>
      currentLocation.pathname !== nextLocation.pathname && hasUnsavedChanges(),
  );

  useEffect(() => {
    if (blocker.state !== "blocked") return;
    let cancelled = false;
    void confirmUnsavedLeave().then((ok) => {
      if (cancelled) return;
      if (ok) blocker.proceed?.();
      else blocker.reset?.();
    });
    return () => {
      cancelled = true;
    };
  }, [blocker]);

  useEffect(() => {
    const onBeforeUnload = (e: BeforeUnloadEvent) => {
      if (!hasUnsavedChanges()) return;
      e.preventDefault();
      e.returnValue = "";
    };
    window.addEventListener("beforeunload", onBeforeUnload);
    return () => window.removeEventListener("beforeunload", onBeforeUnload);
  }, []);

  const navItems = [
    { id: "overview" as Tab, label: "概览", icon: LayoutDashboard },
    { id: "write" as Tab, label: "写作", icon: PenLine },
    { id: "chapters" as Tab, label: "章节", icon: FileText },
    { id: "memory" as Tab, label: "记忆", icon: Brain },
    { id: "entities" as Tab, label: "状态", icon: Users },
    { id: "wiki" as Tab, label: "百科", icon: BookOpen },
  ];

  return (
    <div className="flex h-full min-h-0 flex-col overflow-hidden">
      <header className="flex shrink-0 items-center gap-4 border-b border-studio-border px-5 py-3">
        <Link
          to="/"
          className="inline-flex items-center gap-1 rounded-lg px-2 py-1.5 text-sm text-studio-muted hover:bg-studio-panel hover:text-studio-text"
        >
          <ArrowLeft className="h-4 w-4" />
          书库
        </Link>

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
              <Link
                to="/"
                className="block px-4 py-2 text-sm text-studio-muted hover:text-studio-text"
                onClick={() => setSwitcherOpen(false)}
              >
                返回书库…
              </Link>
            </div>
          )}
        </div>

        <span className="rounded-full bg-studio-accent/10 px-2.5 py-0.5 text-xs text-studio-accent">
          {phaseLabel[status?.phase || ""] || status?.phase}
        </span>

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
          <nav className="space-y-1">
            {navItems.map(({ id, label, icon: Icon }) => (
              <button
                key={id}
                type="button"
                onClick={() => switchTab(id)}
                className={`flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm transition ${
                  tab === id
                    ? "bg-studio-accent/15 text-studio-accent"
                    : "text-studio-muted hover:bg-studio-panel hover:text-studio-text"
                }`}
              >
                <Icon className="h-4 w-4" />
                {label}
              </button>
            ))}
          </nav>
          <p className="mt-8 px-3 text-xs leading-relaxed text-studio-muted">
            状态页查看人物/地点/物品的当前状态；百科查阅设定与大纲；章节页可与改稿顾问讨论。
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
            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
              <ProgressPanel status={status} />
              <StatCard
                label="已写章节"
                value={status.chapter_count}
                accent
                onClick={status.chapter_count > 0 ? goToChapters : undefined}
                hint={status.chapter_count > 0 ? "查看章节列表" : undefined}
              />
              <StatCard
                label="当前章号"
                value={status.current_chapter}
                onClick={status.current_chapter > 0 ? goToCurrentChapter : undefined}
                hint={status.current_chapter > 0 ? "阅读当前章" : undefined}
              />
              <StatCard
                label="记忆条目"
                value={status.memory_count}
                onClick={status.memory_count > 0 ? goToMemories : undefined}
                hint={status.memory_count > 0 ? "查看记忆" : undefined}
              />
              <StatCard
                label="Open 伏笔"
                value={status.open_foreshadows}
                onClick={status.open_foreshadows > 0 ? goToForeshadows : undefined}
                hint={status.open_foreshadows > 0 ? "查看 open 伏笔" : undefined}
              />
              {status.next_steps && status.next_steps.length > 0 && (
                <div className="md:col-span-2 xl:col-span-3 rounded-xl border border-studio-border bg-studio-panel p-5">
                  <h3 className="text-sm font-medium text-studio-muted">建议下一步</h3>
                  <ul className="mt-3 space-y-2">
                    {status.next_steps.map((s) => (
                      <li key={s} className="flex items-center gap-2 text-sm">
                        <PenLine className="h-4 w-4 shrink-0 text-studio-accent" />
                        {s}
                      </li>
                    ))}
                  </ul>
                </div>
              )}
              <ProjectHealthPanel
                refreshKey={healthRefreshKey}
                onPlanVolume={focusPlanVolume}
                onOpenWrite={goToWrite}
                onRebuildIndex={handleRebuildIndex}
                onOpenChapterReview={(num) => void guardedLoadChapter(num, "review")}
              />
              <VolumePlanPanel
                suggestedVolume={status.current_volume || 1}
                focusVolume={planFocusVolume}
                onComplete={handlePlanComplete}
              />
            </div>
            </div>
          )}

          {tab === "write" && (
            <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
              <WritePanel status={status} onComplete={loadStudio} />
            </div>
          )}

          {tab === "chapters" && (
            <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
            <div className="flex min-h-0 flex-1 gap-4 overflow-hidden">
              <div className="flex w-52 shrink-0 flex-col overflow-hidden rounded-xl border border-studio-border bg-studio-panel">
                <ul className="min-h-0 flex-1 overflow-y-auto">
                  {chapters.length === 0 ? (
                    <li className="p-4 text-sm text-studio-muted">暂无章节</li>
                  ) : (
                    chapters.map((c) => (
                      <li key={c.number}>
                        <button
                          type="button"
                          onClick={() => void guardedLoadChapter(c.number)}
                          className={`w-full border-b border-studio-border px-4 py-3 text-left text-sm transition hover:bg-studio-bg ${
                            selectedChapter === c.number ? "bg-studio-bg text-studio-accent" : ""
                          }`}
                        >
                          <div className="font-medium">第{c.number}章</div>
                          <div className="truncate text-xs text-studio-muted">
                            {c.title || "无标题"} · {c.word_count}字
                          </div>
                        </button>
                      </li>
                    ))
                  )}
                </ul>
              </div>
              <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden rounded-xl border border-studio-border bg-studio-paper text-studio-ink">
                {selectedChapter && (
                  <div className="flex shrink-0 items-center justify-between border-b border-studio-border px-4 py-2">
                    <span className="text-sm font-medium">第{selectedChapter}章</span>
                    <button
                      type="button"
                      onClick={() => setVersionPanelOpen(true)}
                      className="inline-flex items-center gap-1 rounded-lg px-2 py-1 text-xs text-studio-muted hover:bg-studio-bg hover:text-studio-text"
                    >
                      <History className="h-3.5 w-3.5" />
                      版本历史
                    </button>
                  </div>
                )}
                <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
                  {selectedChapter ? (
                    <ChapterDocumentPanel
                      key={`${selectedChapter}-${chapterRefreshKey}`}
                      chapter={selectedChapter}
                      initialTab={chapterDocTab}
                      onSaved={refreshChapterView}
                    />
                  ) : (
                    <p className="flex h-full items-center justify-center text-studio-muted/70">
                      选择左侧章节阅读正文
                    </p>
                  )}
                </div>
              </div>
              {selectedChapter && (
                <ChapterCoachPanel
                  chapter={selectedChapter}
                  onApplied={refreshChapterView}
                />
              )}
              {selectedChapter && (
                <ChapterVersionPanel
                  chapter={selectedChapter}
                  open={versionPanelOpen}
                  onClose={() => setVersionPanelOpen(false)}
                  onRestored={refreshChapterView}
                />
              )}
            </div>
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

          {tab === "wiki" && (
            <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
              <WikiPanel initialSelectedID={wikiSelectedID} />
            </div>
          )}

          {tab === "entities" && (
            <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
              <EntityPanel />
            </div>
          )}
        </main>
      </div>

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

function StatCard({
  label,
  value,
  accent,
  onClick,
  hint,
}: {
  label: string;
  value: number;
  accent?: boolean;
  onClick?: () => void;
  hint?: string;
}) {
  const clickable = value > 0 && onClick;
  const inner = (
    <>
      <p className="text-xs uppercase tracking-wide text-studio-muted">{label}</p>
      <p className={`mt-2 text-3xl font-semibold ${accent ? "text-studio-accent" : ""}`}>
        {value}
      </p>
      {clickable && hint && (
        <p className="mt-2 text-xs text-studio-accent/80 opacity-0 transition group-hover:opacity-100">
          {hint} →
        </p>
      )}
    </>
  );

  if (clickable) {
    return (
      <button
        type="button"
        onClick={onClick}
        className="group rounded-xl border border-studio-border bg-studio-panel p-5 text-left transition hover:border-studio-accent/40 hover:bg-studio-accent/5 focus:outline-none focus-visible:ring-2 focus-visible:ring-studio-accent/50"
      >
        {inner}
      </button>
    );
  }

  return (
    <div className="rounded-xl border border-studio-border bg-studio-panel p-5">{inner}</div>
  );
}
