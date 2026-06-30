import { useCallback, useEffect, useMemo, useState } from "react";
import {
  BookOpen,
  ChevronDown,
  ChevronRight,
  ExternalLink,
  Loader2,
  RefreshCw,
  Search,
  Users,
} from "lucide-react";
import { WikiContentDTO, WikiEntryDTO, app } from "../lib/wails";
import { confirmUnsavedLeave } from "../lib/unsavedGuard";
import MarkdownEditor from "./MarkdownEditor";

const GROUP_ORDER = ["人物", "设定", "大纲"] as const;

const groupIcon: Record<string, typeof Users> = {
  人物: Users,
  设定: BookOpen,
  大纲: BookOpen,
};

type Props = {
  initialSelectedID?: string;
};

export default function WikiPanel({ initialSelectedID = "" }: Props) {
  const [entries, setEntries] = useState<WikiEntryDTO[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
  const [selectedID, setSelectedID] = useState("");
  const [content, setContent] = useState<WikiContentDTO | null>(null);
  const [contentLoading, setContentLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [collapsedGroups, setCollapsedGroups] = useState<Record<string, boolean>>({});

  const loadEntries = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const list = await app().ListWikiEntries();
      setEntries(list);
      if (list.length > 0) {
        setSelectedID((prev) => prev || list[0].id);
      }
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

  const loadContent = useCallback(async (id: string) => {
    setContentLoading(true);
    setError("");
    try {
      const c = await app().GetWikiContent(id);
      setContent(c);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setContent(null);
    } finally {
      setContentLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!selectedID) {
      setContent(null);
      return;
    }
    loadContent(selectedID);
  }, [selectedID, loadContent]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return entries;
    return entries.filter(
      (e) =>
        e.title.toLowerCase().includes(q) ||
        e.subtitle.toLowerCase().includes(q) ||
        e.group.toLowerCase().includes(q),
    );
  }, [entries, query]);

  const grouped = useMemo(() => {
    const map = new Map<string, WikiEntryDTO[]>();
    for (const g of GROUP_ORDER) {
      map.set(g, []);
    }
    for (const e of filtered) {
      const list = map.get(e.group) ?? [];
      list.push(e);
      map.set(e.group, list);
    }
    return GROUP_ORDER.map((g) => ({ group: g, items: map.get(g) ?? [] })).filter((x) => x.items.length > 0);
  }, [filtered]);

  const queryActive = query.trim().length > 0;

  const toggleGroup = (group: string) => {
    setCollapsedGroups((prev) => ({ ...prev, [group]: !prev[group] }));
  };

  const isGroupCollapsed = (group: string) => {
    if (queryActive) return false;
    return collapsedGroups[group] ?? false;
  };

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

  return (
    <div className="flex min-h-0 flex-1 gap-4 overflow-hidden">
      <div className="flex w-56 shrink-0 flex-col overflow-hidden rounded-xl border border-studio-border bg-studio-panel xl:w-64">
        <div className="flex shrink-0 items-center justify-between border-b border-studio-border px-3 py-2.5">
          <span className="text-sm font-medium">百科条目</span>
          <button
            type="button"
            onClick={loadEntries}
            className="rounded p-1 text-studio-muted hover:bg-studio-bg hover:text-studio-text"
            title="刷新"
          >
            <RefreshCw className={`h-3.5 w-3.5 ${loading ? "animate-spin" : ""}`} />
          </button>
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

        <div className="min-h-0 flex-1 overflow-y-auto">
          {loading ? (
            <div className="flex items-center justify-center py-10 text-studio-muted">
              <Loader2 className="h-5 w-5 animate-spin" />
            </div>
          ) : grouped.length === 0 ? (
            <p className="p-4 text-center text-sm text-studio-muted">暂无条目</p>
          ) : (
            grouped.map(({ group, items }) => {
              const Icon = groupIcon[group] ?? BookOpen;
              const collapsed = isGroupCollapsed(group);
              return (
                <div key={group} className="border-b border-studio-border/60">
                  <button
                    type="button"
                    onClick={() => toggleGroup(group)}
                    className="sticky top-0 z-10 flex w-full items-center gap-1.5 bg-studio-panel/95 px-3 py-2 text-left text-xs font-medium text-studio-muted backdrop-blur-sm transition hover:bg-studio-bg hover:text-studio-text"
                  >
                    {collapsed ? (
                      <ChevronRight className="h-3.5 w-3.5 shrink-0" />
                    ) : (
                      <ChevronDown className="h-3.5 w-3.5 shrink-0" />
                    )}
                    <Icon className="h-3.5 w-3.5 shrink-0" />
                    <span className="flex-1">{group}</span>
                    <span className="text-studio-muted/60">({items.length})</span>
                  </button>
                  {!collapsed && (
                    <ul>
                      {items.map((e) => (
                        <li key={e.id}>
                          <button
                            type="button"
                            onClick={() => selectEntry(e.id)}
                            className={`w-full border-b border-studio-border/50 px-3 py-2.5 pl-8 text-left text-sm transition hover:bg-studio-bg ${
                              selectedID === e.id ? "bg-studio-bg text-studio-accent" : ""
                            }`}
                          >
                            <div className="truncate font-medium">{e.title}</div>
                            <div className="truncate text-xs text-studio-muted">{e.subtitle}</div>
                          </button>
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              );
            })
          )}
        </div>
      </div>

      <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden rounded-xl border border-studio-border bg-studio-panel">
        {error && !content && (
          <div className="shrink-0 border-b border-studio-border p-3">
            <div className="studio-alert-error-compact">{error}</div>
          </div>
        )}

        {!selectedID ? (
          <p className="flex flex-1 items-center justify-center text-studio-muted/70">选择左侧条目查看详情</p>
        ) : contentLoading ? (
          <div className="flex flex-1 items-center justify-center text-studio-muted">
            <Loader2 className="h-6 w-6 animate-spin" />
          </div>
        ) : content ? (
          <>
            <div className="flex shrink-0 items-center justify-between gap-3 border-b border-studio-border px-4 py-3">
              <div className="min-w-0">
                <h2 className="truncate text-base font-medium">{content.title}</h2>
                <p className="text-xs text-studio-muted">
                  {content.group}
                  {!content.editable && " · 只读"}
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
        ) : null}
      </div>
    </div>
  );
}
