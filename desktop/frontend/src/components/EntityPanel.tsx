import { useCallback, useEffect, useMemo, useState } from "react";
import {
  BookOpen,
  Boxes,
  Loader2,
  MapPin,
  Package,
  RefreshCw,
  Search,
  User,
} from "lucide-react";
import { EntityDTO, app } from "../lib/wails";

const typeMeta: Record<
  string,
  { label: string; icon: typeof User; accent: string; chip: string }
> = {
  character: {
    label: "人物",
    icon: User,
    accent: "text-studio-accent",
    chip: "bg-studio-accent/10 text-studio-accent",
  },
  location: {
    label: "地点",
    icon: MapPin,
    accent: "text-[rgb(var(--studio-diff-add-stat))]",
    chip: "bg-[rgb(var(--studio-diff-add-bg))] text-[rgb(var(--studio-diff-add-stat))]",
  },
  item: {
    label: "物品",
    icon: Package,
    accent: "text-studio-ai",
    chip: "bg-studio-ai/10 text-studio-ai",
  },
};

const typeTabs = [
  { id: "", label: "全部" },
  { id: "character", label: "人物" },
  { id: "location", label: "地点" },
  { id: "item", label: "物品" },
] as const;

function metaFor(type: string) {
  return (
    typeMeta[type] ?? {
      label: type,
      icon: Boxes,
      accent: "text-studio-muted",
      chip: "bg-studio-border/40 text-studio-muted",
    }
  );
}

export default function EntityPanel() {
  const [entities, setEntities] = useState<EntityDTO[]>([]);
  const [typeFilter, setTypeFilter] = useState("");
  const [query, setQuery] = useState("");
  const [selectedId, setSelectedId] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const list = await app().ListEntities(typeFilter);
      setEntities(list ?? []);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [typeFilter]);

  useEffect(() => {
    load();
  }, [load]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return entities;
    return entities.filter(
      (e) =>
        e.name.toLowerCase().includes(q) ||
        e.id.toLowerCase().includes(q) ||
        Object.values(e.state).some((v) => v.toLowerCase().includes(q)),
    );
  }, [entities, query]);

  const counts = useMemo(
    () => ({
      total: entities.length,
      character: entities.filter((e) => e.type === "character").length,
      location: entities.filter((e) => e.type === "location").length,
      item: entities.filter((e) => e.type === "item").length,
    }),
    [entities],
  );

  useEffect(() => {
    if (filtered.length === 0) {
      setSelectedId("");
      return;
    }
    if (!filtered.some((e) => e.id === selectedId)) {
      setSelectedId(filtered[0].id);
    }
  }, [filtered, selectedId]);

  const selected = filtered.find((e) => e.id === selectedId) ?? null;

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-hidden">
      <section className="shrink-0 overflow-hidden rounded-2xl border border-studio-border bg-studio-panel shadow-sm">
        <div className="border-b border-studio-border/60 bg-gradient-to-br from-studio-ai/5 via-transparent to-studio-accent/5 px-5 py-4">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <div className="flex items-center gap-2">
                <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-studio-ai/10">
                  <Boxes className="h-4 w-4 text-studio-ai" />
                </div>
                <div>
                  <h2 className="text-base font-semibold text-studio-text">故事实体</h2>
                  <p className="text-xs text-studio-muted">人物、地点、物品的当前状态快照</p>
                </div>
              </div>
            </div>
            {!loading && entities.length > 0 && (
              <div className="flex flex-wrap gap-2 text-xs">
                <StatChip label="全部" value={counts.total} active={typeFilter === ""} />
                {counts.character > 0 && (
                  <StatChip label="人物" value={counts.character} active={typeFilter === "character"} />
                )}
                {counts.location > 0 && (
                  <StatChip label="地点" value={counts.location} active={typeFilter === "location"} />
                )}
                {counts.item > 0 && (
                  <StatChip label="物品" value={counts.item} active={typeFilter === "item"} />
                )}
              </div>
            )}
          </div>
        </div>
        <p className="px-5 py-3 text-xs leading-relaxed text-studio-muted">
          审查章节后会自动从正文提取并更新。写作时 AI 会参考这些状态，保证人物位置、关系、物品归属前后一致。
        </p>
      </section>

      {error && <div className="shrink-0 studio-alert-error-compact">{error}</div>}

      <div className="flex min-h-0 flex-1 gap-4 overflow-hidden">
        <aside className="flex w-56 shrink-0 flex-col overflow-hidden rounded-xl border border-studio-border bg-studio-panel xl:w-64">
          <div className="flex shrink-0 items-center justify-between border-b border-studio-border px-3 py-2.5">
            <span className="text-sm font-medium">实体列表</span>
            <button
              type="button"
              onClick={load}
              className="rounded p-1 text-studio-muted hover:bg-studio-bg hover:text-studio-text"
              title="刷新"
            >
              <RefreshCw className={`h-3.5 w-3.5 ${loading ? "animate-spin" : ""}`} />
            </button>
          </div>

          <div className="shrink-0 space-y-2 border-b border-studio-border px-3 py-2">
            <div className="flex gap-1 rounded-lg border border-studio-border bg-studio-bg p-0.5">
              {typeTabs.map(({ id, label }) => (
                <button
                  key={id || "all"}
                  type="button"
                  onClick={() => setTypeFilter(id)}
                  className={`flex-1 rounded-md px-1.5 py-1 text-[10px] transition ${
                    typeFilter === id
                      ? "bg-studio-panel text-studio-accent shadow-sm"
                      : "text-studio-muted hover:text-studio-text"
                  }`}
                >
                  {label}
                </button>
              ))}
            </div>
            <div className="flex items-center gap-2 rounded-lg border border-studio-border bg-studio-bg px-2.5 py-1.5">
              <Search className="h-3.5 w-3.5 shrink-0 text-studio-muted" />
              <input
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="搜索名称或状态…"
                className="min-w-0 flex-1 bg-transparent text-sm outline-none placeholder:text-studio-muted/70"
              />
            </div>
          </div>

          <div className="min-h-0 flex-1 overflow-y-auto">
            {loading ? (
              <div className="flex items-center justify-center py-10 text-studio-muted">
                <Loader2 className="h-5 w-5 animate-spin" />
              </div>
            ) : filtered.length === 0 ? (
              <p className="p-4 text-center text-xs leading-relaxed text-studio-muted">
                {entities.length === 0 ? "暂无实体" : "无匹配结果"}
              </p>
            ) : (
              <ul>
                {filtered.map((e) => {
                  const meta = metaFor(e.type);
                  const Icon = meta.icon;
                  const active = e.id === selectedId;
                  return (
                    <li key={e.id}>
                      <button
                        type="button"
                        onClick={() => setSelectedId(e.id)}
                        className={`flex w-full items-start gap-2.5 border-b border-studio-border/60 px-3 py-2.5 text-left transition ${
                          active
                            ? "bg-studio-accent/10 text-studio-accent"
                            : "hover:bg-studio-bg hover:text-studio-text"
                        }`}
                      >
                        <span
                          className={`mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-lg ${
                            active ? "bg-studio-accent/15" : "bg-studio-bg"
                          }`}
                        >
                          <Icon className={`h-3.5 w-3.5 ${active ? "text-studio-accent" : meta.accent}`} />
                        </span>
                        <span className="min-w-0 flex-1">
                          <span className="block truncate text-sm font-medium">{e.name}</span>
                          <span className="mt-0.5 block truncate text-[10px] text-studio-muted">
                            {meta.label}
                            {e.last_chapter > 0 && ` · 第 ${e.last_chapter} 章`}
                          </span>
                        </span>
                      </button>
                    </li>
                  );
                })}
              </ul>
            )}
          </div>
        </aside>

        <main className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden rounded-xl border border-studio-border bg-studio-panel">
          {loading ? (
            <div className="flex flex-1 items-center justify-center text-studio-muted">
              <Loader2 className="h-6 w-6 animate-spin" />
            </div>
          ) : !selected ? (
            <EmptyState hasEntities={entities.length > 0} filteredEmpty={entities.length > 0 && filtered.length === 0} />
          ) : (
            <EntityDetail entity={selected} />
          )}
        </main>
      </div>
    </div>
  );
}

function StatChip({ label, value, active }: { label: string; value: number; active?: boolean }) {
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full px-2.5 py-1 tabular-nums ${
        active ? "bg-studio-accent/15 text-studio-accent" : "bg-studio-bg text-studio-muted"
      }`}
    >
      <span>{label}</span>
      <span className="font-semibold">{value}</span>
    </span>
  );
}

function EmptyState({
  hasEntities,
  filteredEmpty,
}: {
  hasEntities: boolean;
  filteredEmpty: boolean;
}) {
  if (filteredEmpty) {
    return (
      <div className="flex flex-1 flex-col items-center justify-center px-6 text-center">
        <Search className="mb-3 h-8 w-8 text-studio-muted/40" />
        <p className="text-sm text-studio-muted">没有符合筛选条件的实体，试试调整搜索或类型。</p>
      </div>
    );
  }

  return (
    <div className="flex flex-1 flex-col items-center justify-center px-8 text-center">
      <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-studio-bg">
        <Boxes className="h-7 w-7 text-studio-muted/50" />
      </div>
      <h3 className="text-base font-medium text-studio-text">还没有实体状态</h3>
      <p className="mt-2 max-w-sm text-sm leading-relaxed text-studio-muted">
        实体会在写章并完成审查后，从正文自动提取。它们记录人物处境、地点变化、物品归属等结构化信息。
      </p>
      <ol className="mt-6 max-w-xs space-y-2 text-left text-xs text-studio-muted">
        <li className="flex gap-2">
          <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-studio-accent/15 text-[10px] font-medium text-studio-accent">
            1
          </span>
          <span>在「写作」页完成一章正文</span>
        </li>
        <li className="flex gap-2">
          <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-studio-accent/15 text-[10px] font-medium text-studio-accent">
            2
          </span>
          <span>在「章节」页对该章执行 AI 审查</span>
        </li>
        <li className="flex gap-2">
          <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-studio-accent/15 text-[10px] font-medium text-studio-accent">
            3
          </span>
          <span>审查完成后自动沉淀记忆与实体状态</span>
        </li>
      </ol>
      {!hasEntities && (
        <p className="mt-6 flex items-center gap-1.5 text-xs text-studio-muted/80">
          <BookOpen className="h-3.5 w-3.5" />
          同一数据也可在「百科 → 人物」中查看
        </p>
      )}
    </div>
  );
}

function EntityDetail({ entity }: { entity: EntityDTO }) {
  const meta = metaFor(entity.type);
  const Icon = meta.icon;
  const stateEntries = Object.entries(entity.state);
  const initial = entity.name.trim().charAt(0) || "?";

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
      <div className="shrink-0 border-b border-studio-border bg-gradient-to-br from-studio-bg/40 to-transparent px-6 py-5">
        <div className="flex flex-wrap items-start gap-4">
          <div
            className={`flex h-14 w-14 shrink-0 items-center justify-center rounded-2xl text-xl font-semibold ${meta.chip}`}
          >
            {entity.type === "character" ? initial : <Icon className="h-6 w-6" />}
          </div>
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center gap-2">
              <h3 className="text-xl font-semibold text-studio-text">{entity.name}</h3>
              <span className={`rounded-full px-2.5 py-0.5 text-[10px] font-medium ${meta.chip}`}>
                {meta.label}
              </span>
            </div>
            {entity.last_chapter > 0 && (
              <p className="mt-1.5 text-sm text-studio-muted">最近更新于第 {entity.last_chapter} 章</p>
            )}
          </div>
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto px-6 py-5">
        {stateEntries.length > 0 ? (
          <>
            <h4 className="mb-3 text-xs font-medium uppercase tracking-wide text-studio-muted">当前状态</h4>
            <dl className="grid gap-3 sm:grid-cols-2">
              {stateEntries.map(([k, v]) => (
                <div
                  key={k}
                  className="rounded-xl border border-studio-border bg-studio-bg/50 px-4 py-3 transition hover:border-studio-border hover:bg-studio-bg"
                >
                  <dt className="text-[10px] font-medium uppercase tracking-wide text-studio-muted">{k}</dt>
                  <dd className="mt-1.5 text-sm leading-relaxed text-studio-text">{v}</dd>
                </div>
              ))}
            </dl>
          </>
        ) : (
          <div className="flex flex-col items-center justify-center py-12 text-center">
            <p className="text-sm text-studio-muted">暂无结构化状态字段</p>
            <p className="mt-1 max-w-xs text-xs text-studio-muted/70">
              后续章节审查时，若正文涉及该实体，会自动补充状态信息。
            </p>
          </div>
        )}

        <p className="mt-8 flex items-center gap-1.5 text-xs text-studio-muted/70">
          <BookOpen className="h-3.5 w-3.5 shrink-0" />
          完整条目可在「百科 → 人物」中查看；全局搜索也会定位到百科。
        </p>
      </div>
    </div>
  );
}
