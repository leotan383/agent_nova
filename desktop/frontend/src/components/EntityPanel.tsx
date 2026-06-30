import { useCallback, useEffect, useMemo, useState } from "react";
import { Loader2, MapPin, Package, Search, User } from "lucide-react";
import { EntityDTO, app } from "../lib/wails";

const typeTabs = [
  { id: "", label: "全部", icon: Search },
  { id: "character", label: "人物", icon: User },
  { id: "location", label: "地点", icon: MapPin },
  { id: "item", label: "物品", icon: Package },
] as const;

const typeLabel: Record<string, string> = {
  character: "人物",
  location: "地点",
  item: "物品",
};

export default function EntityPanel() {
  const [entities, setEntities] = useState<EntityDTO[]>([]);
  const [typeFilter, setTypeFilter] = useState("");
  const [query, setQuery] = useState("");
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

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
      <div className="mb-4 flex flex-wrap items-center gap-2">
        <div className="flex gap-1 rounded-lg border border-studio-border bg-studio-bg p-1">
          {typeTabs.map(({ id, label, icon: Icon }) => (
            <button
              key={id || "all"}
              type="button"
              onClick={() => setTypeFilter(id)}
              className={`inline-flex items-center gap-1 rounded-md px-2.5 py-1 text-xs transition ${
                typeFilter === id
                  ? "bg-studio-panel text-studio-accent shadow-sm"
                  : "text-studio-muted hover:text-studio-text"
              }`}
            >
              <Icon className="h-3 w-3" />
              {label}
            </button>
          ))}
        </div>
        <div className="relative min-w-[12rem] flex-1">
          <Search className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-studio-muted" />
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="搜索名称或状态…"
            className="w-full rounded-lg border border-studio-border bg-studio-bg py-1.5 pl-8 pr-3 text-xs outline-none focus:border-studio-accent/50"
          />
        </div>
      </div>

      {error && <div className="mb-4 studio-alert-error-compact">{error}</div>}

      {loading ? (
        <div className="flex flex-1 items-center justify-center text-studio-muted">
          <Loader2 className="h-5 w-5 animate-spin" />
        </div>
      ) : filtered.length === 0 ? (
        <p className="text-sm text-studio-muted">
          暂无实体状态。写章并完成「沉淀记忆」后会从正文自动提取人物/地点/物品状态。
        </p>
      ) : (
        <ul className="min-h-0 flex-1 space-y-3 overflow-y-auto">
          {filtered.map((e) => (
            <li key={e.id} className="rounded-xl border border-studio-border bg-studio-panel p-4">
              <div className="flex flex-wrap items-center gap-2">
                <span className="font-medium">{e.name}</span>
                <span className="rounded bg-studio-border px-2 py-0.5 text-[10px] text-studio-muted">
                  {typeLabel[e.type] || e.type}
                </span>
                {e.last_chapter > 0 && (
                  <span className="text-xs text-studio-muted">更新于第 {e.last_chapter} 章</span>
                )}
              </div>
              {Object.keys(e.state).length > 0 ? (
                <dl className="mt-3 grid gap-2 sm:grid-cols-2">
                  {Object.entries(e.state).map(([k, v]) => (
                    <div key={k} className="rounded-lg bg-studio-bg px-3 py-2">
                      <dt className="text-[10px] uppercase tracking-wide text-studio-muted">{k}</dt>
                      <dd className="mt-0.5 text-sm leading-relaxed">{v}</dd>
                    </div>
                  ))}
                </dl>
              ) : (
                <p className="mt-2 text-xs text-studio-muted">无状态字段</p>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
