import { useCallback, useEffect, useState } from "react";
import {
  AlertCircle,
  ArrowRight,
  CheckCircle2,
  Loader2,
  RefreshCw,
  Wrench,
} from "lucide-react";
import { ProjectHealthDTO, TodoItemDTO, app } from "../lib/wails";

type Props = {
  onPlanVolume: (volume: number) => void;
  onOpenWrite: (chapter?: number) => void;
  onRebuildIndex: () => Promise<void>;
  onReviewChapter: (chapter: number) => void;
  refreshKey?: number;
};

const severityStyles: Record<string, string> = {
  urgent: "border-[rgb(var(--studio-warning-border))] bg-[rgb(var(--studio-warning-bg))]",
  warn: "border-amber-500/30 bg-amber-500/5",
  info: "border-studio-border bg-studio-panel",
};

export default function ProjectHealthPanel({
  onPlanVolume,
  onOpenWrite,
  onRebuildIndex,
  onReviewChapter,
  refreshKey = 0,
}: Props) {
  const [health, setHealth] = useState<ProjectHealthDTO | null>(null);
  const [loading, setLoading] = useState(false);
  const [actionBusy, setActionBusy] = useState("");
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const h = await app().GetProjectHealth();
      setHealth(h);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setHealth(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load, refreshKey]);

  const runAction = async (todo: TodoItemDTO) => {
    setActionBusy(todo.id);
    try {
      switch (todo.action) {
        case "plan_volume":
          onPlanVolume(parseInt(todo.action_param || "1", 10) || 1);
          break;
        case "open_write":
          onOpenWrite(todo.action_param ? parseInt(todo.action_param, 10) : undefined);
          break;
        case "rebuild_index":
          await onRebuildIndex();
          await load();
          break;
        case "review_chapter":
          onReviewChapter(parseInt(todo.action_param || "1", 10) || 1);
          break;
        case "open_chapter_review":
          onReviewChapter(parseInt(todo.action_param || "1", 10) || 1);
          break;
        default:
          break;
      }
    } finally {
      setActionBusy("");
    }
  };

  const actionLabel = (todo: TodoItemDTO) => {
    switch (todo.action) {
      case "plan_volume":
        return "生成卷纲";
      case "open_write":
        return "去写作";
      case "rebuild_index":
        return "重建索引";
      case "review_chapter":
        return "开始审查";
      case "open_chapter_review":
        return "开始审查";
      default:
        return "";
    }
  };

  const todos = health?.todos ?? [];
  const urgentCount = todos.filter((t) => t.severity === "urgent").length;

  return (
    <div className="md:col-span-2 xl:col-span-3 rounded-xl border border-studio-border bg-studio-panel p-5">
      <div className="mb-4 flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <h3 className="text-sm font-medium">待办清单</h3>
          {loading ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin text-studio-muted" />
          ) : health?.ok ? (
            <span className="inline-flex items-center gap-1 rounded-full bg-[rgb(var(--studio-diff-add-bg))] px-2 py-0.5 text-xs text-[rgb(var(--studio-diff-add-stat))]">
              <CheckCircle2 className="h-3 w-3" />
              无紧急项
            </span>
          ) : health ? (
            <span className="inline-flex items-center gap-1 rounded-full bg-[rgb(var(--studio-warning-bg))] px-2 py-0.5 text-xs text-[rgb(var(--studio-warning-fg))]">
              <AlertCircle className="h-3 w-3" />
              {urgentCount} 项待处理
            </span>
          ) : null}
        </div>
        <button
          type="button"
          onClick={load}
          disabled={loading}
          className="inline-flex items-center gap-1 rounded-lg px-2 py-1 text-xs text-studio-muted hover:bg-studio-bg hover:text-studio-text"
        >
          <RefreshCw className="h-3 w-3" />
          刷新
        </button>
      </div>

      {error && <div className="mb-3 studio-alert-error-compact">{error}</div>}

      {todos.length === 0 && !loading && (
        <p className="text-sm text-studio-muted">暂无待办，可以继续创作。</p>
      )}

      <ul className="space-y-2">
        {todos.map((todo) => {
          const label = actionLabel(todo);
          const busy = actionBusy === todo.id;
          return (
            <li
              key={todo.id}
              className={`flex items-start justify-between gap-3 rounded-lg border p-3 ${
                severityStyles[todo.severity] ?? severityStyles.info
              }`}
            >
              <div className="min-w-0 flex-1">
                <p className="text-sm font-medium">{todo.label}</p>
                {todo.detail && (
                  <p className="mt-1 text-xs text-studio-muted">{todo.detail}</p>
                )}
              </div>
              {label && todo.action !== "none" && (
                <button
                  type="button"
                  disabled={busy}
                  onClick={() => runAction(todo)}
                  className="inline-flex shrink-0 items-center gap-1 rounded-lg bg-studio-accent px-3 py-1.5 text-xs font-medium text-white hover:opacity-90 disabled:opacity-50"
                >
                  {busy ? (
                    <Loader2 className="h-3 w-3 animate-spin" />
                  ) : todo.action === "rebuild_index" ? (
                    <Wrench className="h-3 w-3" />
                  ) : (
                    <ArrowRight className="h-3 w-3" />
                  )}
                  {label}
                </button>
              )}
            </li>
          );
        })}
      </ul>
    </div>
  );
}
