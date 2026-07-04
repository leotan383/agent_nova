import { useEffect, useState } from "react";
import { Boxes, Clock, MapPin, Package, User } from "lucide-react";
import { EntityDTO, EntityStateSnapshotDTO, app } from "../lib/wails";
import EntityTimeline from "./EntityTimeline";

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

export function metaFor(type: string) {
  return (
    typeMeta[type] ?? {
      label: type,
      icon: Boxes,
      accent: "text-studio-muted",
      chip: "bg-studio-border/40 text-studio-muted",
    }
  );
}

type ViewMode = "current" | "timeline";

export default function EntityDetailView({ entity }: { entity: EntityDTO }) {
  const meta = metaFor(entity.type);
  const Icon = meta.icon;
  const stateEntries = Object.entries(entity.state).filter(([k]) => k !== "aliases");
  const initial = entity.name.trim().charAt(0) || "?";

  const [view, setView] = useState<ViewMode>("current");
  const [history, setHistory] = useState<EntityStateSnapshotDTO[]>([]);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [historyError, setHistoryError] = useState("");

  useEffect(() => {
    setView("current");
    setHistory([]);
    setHistoryError("");
  }, [entity.id]);

  useEffect(() => {
    if (view !== "timeline") return;
    let cancelled = false;
    setHistoryLoading(true);
    setHistoryError("");
    app()
      .GetEntityHistory(entity.id)
      .then((list) => {
        if (!cancelled) setHistory(list);
      })
      .catch((e) => {
        if (!cancelled) setHistoryError(e instanceof Error ? e.message : String(e));
      })
      .finally(() => {
        if (!cancelled) setHistoryLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [view, entity.id]);

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
              <span className="rounded-full bg-studio-bg px-2 py-0.5 text-[10px] text-studio-muted">
                审查自动更新
              </span>
            </div>
            {entity.last_chapter > 0 && (
              <p className="mt-1.5 text-sm text-studio-muted">最近更新于第 {entity.last_chapter} 章</p>
            )}
          </div>
        </div>

        <div className="mt-4 flex gap-1 rounded-lg border border-studio-border bg-studio-panel/50 p-1">
          {(
            [
              { id: "current" as const, label: "当前状态" },
              { id: "timeline" as const, label: "状态时间线", icon: Clock },
            ] as const
          ).map((tab) => (
            <button
              key={tab.id}
              type="button"
              onClick={() => setView(tab.id)}
              className={`flex min-w-0 flex-1 items-center justify-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition ${
                view === tab.id
                  ? "bg-studio-accent/15 text-studio-accent"
                  : "text-studio-muted hover:bg-studio-bg hover:text-studio-text"
              }`}
            >
              {"icon" in tab && tab.icon && <tab.icon className="h-3.5 w-3.5 shrink-0" />}
              {tab.label}
            </button>
          ))}
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto px-6 py-5">
        {view === "current" ? (
          stateEntries.length > 0 ? (
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
          )
        ) : (
          <>
            {historyError && (
              <div className="mb-4 studio-alert-error-compact">{historyError}</div>
            )}
            <EntityTimeline snapshots={history} loading={historyLoading} accentClass={meta.accent} />
          </>
        )}
      </div>
    </div>
  );
}
