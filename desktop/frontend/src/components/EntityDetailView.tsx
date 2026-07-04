import { useCallback, useEffect, useMemo, useState } from "react";
import { Boxes, Clock, MapPin, Package, User } from "lucide-react";
import { EntityDTO, EntityStateSnapshotDTO, app } from "../lib/wails";
import { ENTITY_HISTORY_EVENTS, eventsOn } from "../lib/runtime";
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
  const [backfillRunning, setBackfillRunning] = useState(false);
  const [backfillMessage, setBackfillMessage] = useState("");
  const [backfillError, setBackfillError] = useState("");

  const loadHistory = useCallback(() => {
    setHistoryLoading(true);
    setHistoryError("");
    return app()
      .GetEntityHistory(entity.id)
      .then((list) => setHistory(list))
      .catch((e) => setHistoryError(e instanceof Error ? e.message : String(e)))
      .finally(() => setHistoryLoading(false));
  }, [entity.id]);

  const needsBackfill = useMemo(() => {
    if (entity.last_chapter <= 1) return false;
    if (history.length === 0) return true;
    if (history.length === 1) return true;
    const firstChapter = history[0]?.chapter ?? 0;
    return firstChapter > 1;
  }, [entity.last_chapter, history]);

  useEffect(() => {
    setView("current");
    setHistory([]);
    setHistoryError("");
    setBackfillError("");
    setBackfillMessage("");
  }, [entity.id]);

  useEffect(() => {
    app()
      .GetActiveEntityHistoryBackfillJob()
      .then((active) => {
        if (active.active) {
          setBackfillRunning(true);
        }
      })
      .catch(() => {});
  }, []);

  useEffect(() => {
    const offStatus = eventsOn(ENTITY_HISTORY_EVENTS.status, (payload) => {
      setBackfillRunning(payload.status === "running" || payload.status === "pending");
      if (payload.message) setBackfillMessage(payload.message);
    });
    const offDone = eventsOn(ENTITY_HISTORY_EVENTS.done, (payload) => {
      setBackfillRunning(false);
      const skipped = payload.skipped ?? [];
      if (skipped.length > 0) {
        setBackfillMessage(`回溯完成，${skipped.length} 章跳过`);
        setBackfillError(skipped.join("；"));
      } else {
        setBackfillMessage("回溯完成");
        setBackfillError("");
      }
      if (view === "timeline") void loadHistory();
    });
    const offError = eventsOn(ENTITY_HISTORY_EVENTS.error, (payload) => {
      setBackfillRunning(false);
      setBackfillError(payload.error ?? "回溯失败");
    });
    return () => {
      offStatus();
      offDone();
      offError();
    };
  }, [view, loadHistory]);

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

  const handleBackfill = useCallback(() => {
    setBackfillError("");
    setBackfillMessage("准备回溯…");
    app()
      .StartEntityHistoryBackfill()
      .then(() => setBackfillRunning(true))
      .catch((e) => {
        setBackfillRunning(false);
        setBackfillError(e instanceof Error ? e.message : String(e));
      });
  }, []);

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
            <EntityTimeline
              snapshots={history}
              loading={historyLoading}
              accentClass={meta.accent}
              lastChapter={entity.last_chapter}
              needsBackfill={needsBackfill}
              backfillRunning={backfillRunning}
              backfillMessage={backfillMessage}
              backfillError={backfillError}
              onBackfill={handleBackfill}
            />
          </>
        )}
      </div>
    </div>
  );
}
