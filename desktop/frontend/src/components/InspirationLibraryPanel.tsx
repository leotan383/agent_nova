import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Archive, Lightbulb, Plus, Search, SlidersHorizontal, X } from "lucide-react";
import ConfirmDialog from "./ConfirmDialog";
import InspirationCardView from "./InspirationCard";
import InspirationEditor from "./InspirationEditor";
import InspirationQuickCapture from "./InspirationQuickCapture";
import {
  InspirationFilters,
  InspirationSortKey,
  filterInspirations,
  inspirationStats,
  inspirationStatusOptions,
  sortInspirations,
} from "../lib/inspirationUtils";
import { InspirationCard, app, genreOptions } from "../lib/wails";

type ConfirmState = {
  title: string;
  message: string;
  confirmLabel: string;
  destructive?: boolean;
  onConfirm: () => void;
};

type Props = {
  onCreateNovel: (inspirationId: string) => void;
  onOpenNovel: (novelId: string) => void;
};

const sortOptions: { id: InspirationSortKey; label: string }[] = [
  { id: "updated", label: "最近更新" },
  { id: "created", label: "最近创建" },
  { id: "title", label: "标题" },
];

export default function InspirationLibraryPanel({ onCreateNovel, onOpenNovel }: Props) {
  const [items, setItems] = useState<InspirationCard[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [showCapture, setShowCapture] = useState(false);
  const [editingId, setEditingId] = useState("");
  const [sort, setSort] = useState<InspirationSortKey>("updated");
  const [filters, setFilters] = useState<InspirationFilters>({
    query: "",
    genre: "",
    status: "",
    showArchived: false,
  });
  const [filterOpen, setFilterOpen] = useState(false);
  const [confirm, setConfirm] = useState<ConfirmState | null>(null);
  const filterRef = useRef<HTMLDivElement>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const list = await app().ListInspirations({ include_archived: filters.showArchived });
      setItems(list ?? []);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [filters.showArchived]);

  useEffect(() => {
    void refresh();
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

  const archivedCount = useMemo(() => items.filter((i) => i.archived).length, [items]);
  const stats = useMemo(() => inspirationStats(items), [items]);
  const filterCount = (filters.genre ? 1 : 0) + (filters.status ? 1 : 0) + (sort !== "updated" ? 1 : 0);

  const filtered = useMemo(() => {
    return sortInspirations(filterInspirations(items, filters), sort);
  }, [items, filters, sort]);

  const askDelete = (item: InspirationCard) => {
    setConfirm({
      title: "删除灵感",
      message: `确定删除「${item.title}」？此操作不可恢复。`,
      confirmLabel: "删除",
      destructive: true,
      onConfirm: async () => {
        setConfirm(null);
        await app().DeleteInspiration(item.id);
        if (editingId === item.id) setEditingId("");
        await refresh();
      },
    });
  };

  const askArchive = (item: InspirationCard) => {
    const archiving = !item.archived;
    setConfirm({
      title: archiving ? "归档灵感" : "取消归档",
      message: archiving ? `「${item.title}」将移入归档。` : `「${item.title}」将恢复到主列表。`,
      confirmLabel: archiving ? "归档" : "恢复",
      onConfirm: async () => {
        setConfirm(null);
        await app().SetInspirationArchived(item.id, archiving);
        await refresh();
      },
    });
  };

  return (
    <>
      <header className="mb-8 flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-studio-text">灵感库</h1>
          {!loading && stats.count > 0 && (
            <p className="mt-1 text-sm text-studio-muted">
              {stats.count} 条灵感
              {stats.ready > 0 && ` · ${stats.ready} 条可开书`}
            </p>
          )}
        </div>
        <button
          type="button"
          onClick={() => setShowCapture(true)}
          className="inline-flex items-center gap-1.5 rounded-xl bg-studio-accent px-4 py-2 text-sm font-medium text-studio-on-accent hover:brightness-110"
        >
          <Plus className="h-4 w-4" />
          记录灵感
        </button>
      </header>

      {error && !showCapture && !editingId && (
        <div className="mb-6 studio-alert-error-compact">{error}</div>
      )}

      {!loading && items.some((i) => !i.archived || filters.showArchived) && (
        <div className="mb-6 flex gap-2">
          <div className="relative min-w-0 flex-1">
            <Search className="pointer-events-none absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-studio-muted/70" />
            <input
              value={filters.query}
              onChange={(e) => setFilters((f) => ({ ...f, query: e.target.value }))}
              placeholder="搜索灵感…"
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
                <label className="mb-1 block text-[11px] text-studio-muted">状态</label>
                <select
                  value={filters.status}
                  onChange={(e) => setFilters((f) => ({ ...f, status: e.target.value }))}
                  className="mb-3 w-full rounded-lg border border-studio-border bg-studio-bg px-2.5 py-1.5 text-sm outline-none"
                >
                  {inspirationStatusOptions.map((o) => (
                    <option key={o.value || "all"} value={o.value}>
                      {o.label}
                    </option>
                  ))}
                </select>
                <label className="mb-1 block text-[11px] text-studio-muted">排序</label>
                <select
                  value={sort}
                  onChange={(e) => setSort(e.target.value as InspirationSortKey)}
                  className="w-full rounded-lg border border-studio-border bg-studio-bg px-2.5 py-1.5 text-sm outline-none"
                >
                  {sortOptions.map((o) => (
                    <option key={o.id} value={o.id}>
                      {o.label}
                    </option>
                  ))}
                </select>
              </div>
            )}
          </div>
        </div>
      )}

      {loading ? (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-52 animate-pulse rounded-2xl bg-studio-panel ring-1 ring-studio-border/60" />
          ))}
        </div>
      ) : filters.showArchived ? (
        filtered.length === 0 ? (
          <p className="py-20 text-center text-sm text-studio-muted">没有已归档的灵感</p>
        ) : (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {filtered.map((item) => (
              <InspirationCardView
                key={item.id}
                item={item}
                onEdit={() => setEditingId(item.id)}
                onCreateNovel={() => onCreateNovel(item.id)}
                onTogglePin={async () => {
                  await app().SetInspirationPinned(item.id, !item.pinned);
                  await refresh();
                }}
                onToggleArchive={() => askArchive(item)}
                onDelete={() => askDelete(item)}
                onOpenNovel={item.novel_id ? () => onOpenNovel(item.novel_id!) : undefined}
              />
            ))}
          </div>
        )
      ) : items.filter((i) => !i.archived).length === 0 ? (
        <div className="flex flex-col items-center py-20 text-center">
          <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-studio-accent/10">
            <Lightbulb className="h-7 w-7 text-studio-accent" />
          </div>
          <h2 className="text-lg font-medium">收集你的创意火花</h2>
          <p className="mt-2 max-w-md text-sm leading-relaxed text-studio-muted">
            随时记录灵感，独立于具体的书。想开新书时，从灵感库一键预填立项信息。
          </p>
          <button
            type="button"
            onClick={() => setShowCapture(true)}
            className="mt-8 rounded-xl bg-studio-accent px-5 py-2 text-sm font-medium text-studio-on-accent"
          >
            记录第一条灵感
          </button>
        </div>
      ) : filtered.length === 0 ? (
        <div className="py-20 text-center">
          <p className="text-sm text-studio-muted">没有匹配的灵感</p>
          <button
            type="button"
            onClick={() => setFilters((f) => ({ ...f, query: "", genre: "", status: "" }))}
            className="mt-2 text-sm text-studio-accent hover:underline"
          >
            清除搜索
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {filtered.map((item) => (
            <InspirationCardView
              key={item.id}
              item={item}
              onEdit={() => setEditingId(item.id)}
              onCreateNovel={() => onCreateNovel(item.id)}
              onTogglePin={async () => {
                await app().SetInspirationPinned(item.id, !item.pinned);
                await refresh();
              }}
              onToggleArchive={() => askArchive(item)}
              onDelete={() => askDelete(item)}
              onOpenNovel={item.novel_id ? () => onOpenNovel(item.novel_id!) : undefined}
            />
          ))}
        </div>
      )}

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
                status: "",
              }))
            }
            className="inline-flex items-center gap-1.5 text-xs text-studio-muted transition hover:text-studio-text"
          >
            <Archive className="h-3.5 w-3.5" />
            {filters.showArchived ? "返回灵感库" : `已归档 · ${archivedCount}`}
          </button>
        </div>
      )}

      {showCapture && (
        <div className="studio-modal-overlay fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="w-full max-w-lg rounded-2xl border border-studio-border bg-studio-panel p-6 shadow-card">
            <h2 className="text-lg font-medium">记录灵感</h2>
            <p className="mt-1 text-sm text-studio-muted">写下大致想法即可，细节以后再补。</p>
            <div className="mt-4">
              <InspirationQuickCapture
                onSaved={async () => {
                  setShowCapture(false);
                  await refresh();
                }}
                onCancel={() => setShowCapture(false)}
              />
            </div>
          </div>
        </div>
      )}

      {editingId && (
        <div className="studio-modal-overlay fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-2xl border border-studio-border bg-studio-panel p-6 shadow-card">
            <InspirationEditor
              id={editingId}
              onClose={() => setEditingId("")}
              onCreateNovel={() => {
                setEditingId("");
                onCreateNovel(editingId);
              }}
              onSaved={() => void refresh()}
              onDeleted={() => {
                setEditingId("");
                void refresh();
              }}
            />
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
    </>
  );
}
