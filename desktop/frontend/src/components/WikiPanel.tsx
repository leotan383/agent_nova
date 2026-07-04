import { useCallback, useEffect, useMemo, useState } from "react";
import {
  BookOpen,
  CircleHelp,
  ExternalLink,
  Loader2,
  Plus,
  RefreshCw,
  Search,
} from "lucide-react";
import { EntityDTO, SettingCategoryDTO, StatusReport, WikiContentDTO, WikiEntryDTO, app } from "../lib/wails";
import {
  META_SYNOPSIS_ID,
  SettingCategory,
  buildSubdirToCategory,
  categoryLabel,
  filterByCategory,
  splitCategoryEntries,
} from "../lib/wikiCategories";
import { settingCategoryIcon } from "../lib/settingCategoryIcons";
import { confirmUnsavedLeave } from "../lib/unsavedGuard";
import EntityDetailView from "./EntityDetailView";
import CreateSettingDialog from "./CreateSettingDialog";
import MarkdownEditor from "./MarkdownEditor";
import SettingTemplateChecklist from "./SettingTemplateChecklist";

const kindLabel: Record<string, string> = {
  setting: "设定集",
  outline: "大纲",
  entity: "AI提取",
  memory: "记忆",
  meta: "作品信息",
};

type Props = {
  status?: StatusReport | null;
  initialSelectedID?: string;
  settingCategories: SettingCategoryDTO[];
  /** 从主导航进入某一设定分类时，只展示该分类下的条目 */
  categoryFilter?: SettingCategory | null;
  /** 从主导航进入简介/大纲时，只展示主题区 */
  themeOnly?: boolean;
  checklistRefreshKey?: number;
  onCreateChecklistItem?: (category: string, title: string, templateKind: string) => void;
  onOpenChecklistSetting?: (wikiID: string) => void;
};

function parseEntityID(wikiID: string): string {
  const i = wikiID.indexOf(":");
  return i >= 0 ? wikiID.slice(i + 1) : wikiID;
}

type CategorySubview = "settings" | "states";

const CATEGORY_SUBVIEW_HINTS: Partial<Record<CategorySubview, string>> = {
  states:
    "每章审查完成后，系统从正文与摘要中自动提取角色/地点/物品等信息，存入项目数据库并随连载更新。与「设定」中的手写档案不同，此处为只读。",
};

export default function WikiPanel({
  status,
  initialSelectedID = "",
  settingCategories,
  categoryFilter = null,
  themeOnly = false,
  checklistRefreshKey = 0,
  onCreateChecklistItem,
  onOpenChecklistSetting,
}: Props) {
  const [entries, setEntries] = useState<WikiEntryDTO[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
  const [selectedID, setSelectedID] = useState("");
  const [content, setContent] = useState<WikiContentDTO | null>(null);
  const [entityData, setEntityData] = useState<EntityDTO | null>(null);
  const [contentLoading, setContentLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [categorySubview, setCategorySubview] = useState<CategorySubview>("settings");
  const [createOpen, setCreateOpen] = useState(false);

  const loadEntries = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      await app().MergeEntityDuplicates();
      const list = await app().ListWikiEntries();
      setEntries(list);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadEntries();
  }, [loadEntries]);

  useEffect(() => {
    if (initialSelectedID) {
      setSelectedID(initialSelectedID);
    }
  }, [initialSelectedID]);

  const subdirToCategory = useMemo(
    () => buildSubdirToCategory(settingCategories),
    [settingCategories],
  );

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    let list = entries;
    if (categoryFilter) {
      list = filterByCategory(list, categoryFilter, subdirToCategory);
    }
    if (!q) return list;
    return list.filter(
      (e) =>
        e.title.toLowerCase().includes(q) ||
        e.subtitle.toLowerCase().includes(q) ||
        (kindLabel[e.kind] ?? e.kind).toLowerCase().includes(q),
    );
  }, [entries, query, categoryFilter, subdirToCategory]);

  const { settingEntries, entityEntries } = useMemo(() => {
    if (!categoryFilter) {
      return { settingEntries: [] as WikiEntryDTO[], entityEntries: [] as WikiEntryDTO[] };
    }
    const { archives, states } = splitCategoryEntries(filtered, categoryFilter, subdirToCategory);
    const byTitle = (a: WikiEntryDTO, b: WikiEntryDTO) =>
      a.title.localeCompare(b.title, "zh-CN");
    return {
      settingEntries: [...archives].sort(byTitle),
      entityEntries: [...states].sort(byTitle),
    };
  }, [filtered, categoryFilter]);

  const activeCategoryEntries =
    categorySubview === "settings" ? settingEntries : entityEntries;

  useEffect(() => {
    setCategorySubview("settings");
  }, [categoryFilter]);

  // 切换分类 / 子 TAB 时，若当前选中不在列表内则自动选中第一条
  useEffect(() => {
    if (loading || themeOnly || initialSelectedID) return;
    if (!categoryFilter) return;
    const list = categorySubview === "settings" ? settingEntries : entityEntries;
    setSelectedID((prev) => {
      if (list.some((e) => e.id === prev)) return prev;
      return list[0]?.id ?? settingEntries[0]?.id ?? entityEntries[0]?.id ?? "";
    });
  }, [
    categoryFilter,
    categorySubview,
    settingEntries,
    entityEntries,
    loading,
    themeOnly,
    initialSelectedID,
  ]);

  const loadContent = useCallback(
    async (id: string) => {
      setContentLoading(true);
      setError("");
      setEntityData(null);

      if (id === META_SYNOPSIS_ID) {
        setContent({
          id: META_SYNOPSIS_ID,
          title: "简介",
          group: "主题",
          kind: "meta",
          body:
            status?.synopsis?.trim() ||
            "暂无简介。可在立项时填写，或编辑 nova.yaml 中的 synopsis 字段。",
          can_open: false,
          editable: false,
        });
        setContentLoading(false);
        return;
      }

      try {
        const c = await app().GetWikiContent(id);
        setContent(c);
        if (c.kind === "entity") {
          const entityId = parseEntityID(c.id);
          const list = await app().ListEntities("");
          const canonicalKey = entityId.includes(":") ? entityId : `character:${entityId}`;
          setEntityData(
            list.find((e) => e.id === entityId || e.id === canonicalKey) ??
              list.find((e) => {
                const name = e.name?.trim() ?? "";
                const canon = name.replace(/[（(][^）)]*[）)]$/, "").trim();
                return `${e.type || "character"}:${canon}` === canonicalKey;
              }) ??
              null,
          );
        }
      } catch (e) {
        setError(e instanceof Error ? e.message : String(e));
        setContent(null);
      } finally {
        setContentLoading(false);
      }
    },
    [status?.synopsis],
  );

  useEffect(() => {
    if (!selectedID) {
      setContent(null);
      setEntityData(null);
      return;
    }
    loadContent(selectedID);
  }, [selectedID, loadContent]);

  const openInFolder = async () => {
    if (!content?.path) return;
    try {
      await app().RevealInFolder(content.path);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const saveContent = async (body: string) => {
    if (!selectedID) return;
    setSaving(true);
    try {
      await app().SaveWikiContent(selectedID, body);
      setContent((prev) => (prev ? { ...prev, body } : prev));
    } finally {
      setSaving(false);
    }
  };

  const selectEntry = async (id: string) => {
    if (id === selectedID) return;
    const ok = await confirmUnsavedLeave();
    if (!ok) return;
    setSelectedID(id);
  };

  const switchCategorySubview = async (next: CategorySubview) => {
    if (next === categorySubview) return;
    const ok = await confirmUnsavedLeave();
    if (!ok) return;
    setCategorySubview(next);
    const list = next === "settings" ? settingEntries : entityEntries;
    if (list.length === 0) {
      setSelectedID("");
      return;
    }
    if (!list.some((e) => e.id === selectedID)) {
      setSelectedID(list[0].id);
    }
  };

  const openCreateSetting = () => {
    if (!categoryFilter) return;
    setCreateOpen(true);
  };

  const handleSettingCreated = async (id: string) => {
    setCategorySubview("settings");
    await loadEntries();
    setSelectedID(id);
  };

  const showEntityView = content?.kind === "entity" && entityData;
  const panelTitle = categoryFilter ? categoryLabel(categoryFilter, settingCategories) : "设定";

  const CategoryIcon = categoryFilter ? settingCategoryIcon(categoryFilter) : BookOpen;

  const renderEntryRow = (e: WikiEntryDTO) => (
    <li key={e.id}>
      <button
        type="button"
        onClick={() => selectEntry(e.id)}
        className={`w-full border-b border-studio-border/50 px-3 py-2.5 text-left text-sm transition hover:bg-studio-bg ${
          selectedID === e.id ? "bg-studio-bg text-studio-accent" : ""
        }`}
      >
        <span className="block truncate font-medium">{e.title}</span>
        {e.subtitle && (
          <div className="truncate text-xs text-studio-muted">{e.subtitle}</div>
        )}
      </button>
    </li>
  );

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
      <div className={`flex min-h-0 flex-1 overflow-hidden ${themeOnly ? "" : "gap-4"}`}>
        {!themeOnly && (
        <div className="flex w-56 shrink-0 flex-col overflow-hidden rounded-xl border border-studio-border bg-studio-panel xl:w-64">
          <div className="flex shrink-0 items-center justify-between border-b border-studio-border px-3 py-2.5">
            <div className="flex items-center gap-2 min-w-0">
              <CategoryIcon className="h-4 w-4 shrink-0 text-studio-muted" />
              <span className="truncate text-sm font-medium">{panelTitle}</span>
              {categoryFilter && (settingEntries.length > 0 || entityEntries.length > 0) && (
                <span className="text-xs text-studio-muted">
                  ({settingEntries.length}
                  {entityEntries.length > 0 && ` + ${entityEntries.length}`})
                </span>
              )}
            </div>
            <div className="flex items-center gap-0.5">
              {categoryFilter && categorySubview === "settings" && (
                <button
                  type="button"
                  onClick={openCreateSetting}
                  className="rounded p-1 text-studio-muted hover:bg-studio-bg hover:text-studio-text"
                  title={`新建${categoryFilter}设定`}
                >
                  <Plus className="h-3.5 w-3.5" />
                </button>
              )}
              <button
                type="button"
                onClick={loadEntries}
                className="rounded p-1 text-studio-muted hover:bg-studio-bg hover:text-studio-text"
                title="刷新"
              >
                <RefreshCw className={`h-3.5 w-3.5 ${loading ? "animate-spin" : ""}`} />
              </button>
            </div>
          </div>

          <div className="shrink-0 border-b border-studio-border px-3 py-2">
            <div className="flex items-center gap-2 rounded-lg border border-studio-border bg-studio-bg px-2.5 py-1.5">
              <Search className="h-3.5 w-3.5 shrink-0 text-studio-muted" />
              <input
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="搜索…"
                className="min-w-0 flex-1 bg-transparent text-sm outline-none placeholder:text-studio-muted/70"
              />
            </div>
          </div>

          {categoryFilter && !themeOnly && (
            <div className="flex shrink-0 gap-1 border-b border-studio-border bg-studio-panel/30 px-2 py-1.5">
              {(
                [
                  { id: "settings" as const, label: "设定", count: settingEntries.length },
                  { id: "states" as const, label: "AI提取", count: entityEntries.length },
                ] as const
              ).map(({ id, label, count }) => (
                <button
                  key={id}
                  type="button"
                  onClick={() => void switchCategorySubview(id)}
                  className={`flex min-w-0 flex-1 items-center justify-center gap-1 rounded-md px-2 py-1 text-[11px] font-medium transition ${
                    categorySubview === id
                      ? "bg-studio-accent/15 text-studio-accent"
                      : "text-studio-muted hover:bg-studio-bg hover:text-studio-text"
                  }`}
                >
                  <span className="truncate">{label}</span>
                  {id === "states" && (
                    <span
                      role="img"
                      aria-label="AI提取数据来源说明"
                      title={CATEGORY_SUBVIEW_HINTS.states}
                      className="inline-flex shrink-0 rounded-sm opacity-70 hover:opacity-100"
                      onClick={(e) => e.stopPropagation()}
                      onMouseDown={(e) => e.stopPropagation()}
                    >
                      <CircleHelp className="h-3 w-3" />
                    </span>
                  )}
                  <span className="shrink-0 tabular-nums text-[10px] opacity-80">{count}</span>
                </button>
              ))}
            </div>
          )}

          <div className="min-h-0 flex-1 overflow-y-auto">
            {loading ? (
              <div className="flex items-center justify-center py-10 text-studio-muted">
                <Loader2 className="h-5 w-5 animate-spin" />
              </div>
            ) : categoryFilter ? (
              <>
                {onCreateChecklistItem && (
                  <SettingTemplateChecklist
                    refreshKey={checklistRefreshKey}
                    categoryFilter={categoryFilter}
                    onCreateItem={onCreateChecklistItem}
                    onOpenSetting={onOpenChecklistSetting}
                    className="mx-3 mt-3 shrink-0"
                  />
                )}
                {activeCategoryEntries.length === 0 ? (
                  <p className="px-4 py-6 text-center text-xs leading-relaxed text-studio-muted">
                    {categorySubview === "settings"
                      ? "暂无设定文档，点击右上角 + 在文件夹中新建 Markdown"
                      : "暂无 AI 提取记录，写章审查后会自动更新"}
                  </p>
                ) : (
                  <ul>{activeCategoryEntries.map(renderEntryRow)}</ul>
                )}
              </>
            ) : (
              <p className="px-4 py-6 text-center text-xs text-studio-muted">
                请从左侧选择设定分类
              </p>
            )}
          </div>
        </div>
        )}

        <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden rounded-xl border border-studio-border bg-studio-panel">
          {error && !content && (
            <div className="shrink-0 border-b border-studio-border p-3">
              <div className="studio-alert-error-compact">{error}</div>
            </div>
          )}

          {!selectedID ? (
            <p className="flex flex-1 items-center justify-center text-studio-muted/70">
              {themeOnly ? "请从左侧「创作 → 大纲」选择条目" : "选择条目查看详情"}
            </p>
          ) : contentLoading ? (
            <div className="flex flex-1 items-center justify-center text-studio-muted">
              <Loader2 className="h-6 w-6 animate-spin" />
            </div>
          ) : content ? (
            showEntityView ? (
              <EntityDetailView entity={entityData} />
            ) : (
              <>
                <div className="flex shrink-0 items-center justify-between gap-3 border-b border-studio-border px-4 py-3">
                  <div className="min-w-0">
                    <h2 className="truncate text-base font-medium">{content.title}</h2>
                    <p className="text-xs text-studio-muted">
                      {content.group}
                      {kindLabel[content.kind] && ` · ${kindLabel[content.kind]}`}
                      {!content.editable && content.kind !== "entity" && " · 只读"}
                    </p>
                  </div>
                  {content.can_open && content.path && (
                    <button
                      type="button"
                      onClick={openInFolder}
                      className="inline-flex shrink-0 items-center gap-1 rounded-lg border border-studio-border px-2.5 py-1.5 text-xs text-studio-muted hover:bg-studio-bg hover:text-studio-text"
                    >
                      <ExternalLink className="h-3.5 w-3.5" />
                      打开文件
                    </button>
                  )}
                </div>
                <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
                  <MarkdownEditor
                    key={selectedID}
                    value={content.body}
                    editable={content.editable}
                    saving={saving}
                    onSave={content.editable ? saveContent : undefined}
                    emptyHint="暂无内容"
                  />
                </div>
              </>
            )
          ) : null}
        </div>
      </div>

      {categoryFilter && (
        <CreateSettingDialog
          open={createOpen}
          category={categoryFilter}
          categories={settingCategories}
          onClose={() => setCreateOpen(false)}
          onCreated={(id) => void handleSettingCreated(id)}
        />
      )}
    </div>
  );
}
