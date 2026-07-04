import { useCallback, useEffect, useState } from "react";
import {
  Calendar,
  CheckCircle2,
  ChevronRight,
  Flame,
  Loader2,
  Settings2,
  Target,
} from "lucide-react";
import { DailyWorkflowDTO, TodoItemDTO, app } from "../lib/wails";

type Props = {
  refreshKey: number;
  onOpenWrite: () => void;
  onReviewChapter: (chapter: number) => void;
  onOpenChapters: () => void;
};

export default function DailyWorkflowCard({
  refreshKey,
  onOpenWrite,
  onReviewChapter,
  onOpenChapters,
}: Props) {
  const [data, setData] = useState<DailyWorkflowDTO | null>(null);
  const [loading, setLoading] = useState(true);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [dailyWords, setDailyWords] = useState(0);
  const [dailyChapters, setDailyChapters] = useState(1);
  const [bufferTarget, setBufferTarget] = useState(7);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const w = await app().GetDailyWorkflow();
      setData(w);
      setDailyWords(w.settings.daily_words);
      setDailyChapters(w.settings.daily_chapters || 1);
      setBufferTarget(w.settings.buffer_target || 7);
    } catch {
      setData(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load, refreshKey]);

  const runAction = (todo: TodoItemDTO) => {
    switch (todo.action) {
      case "open_write":
        onOpenWrite();
        break;
      case "review_chapter":
        onReviewChapter(parseInt(todo.action_param || "1", 10) || 1);
        break;
      case "open_chapters":
        onOpenChapters();
        break;
      default:
        break;
    }
  };

  const saveSettings = async () => {
    setSaving(true);
    try {
      await app().UpdateWorkflowSettings({
        daily_words: dailyWords,
        daily_chapters: dailyChapters,
        buffer_target: bufferTarget,
      });
      setSettingsOpen(false);
      await load();
    } finally {
      setSaving(false);
    }
  };

  if (loading && !data) {
    return (
      <section className="flex items-center gap-2 rounded-2xl border border-studio-border bg-studio-panel px-5 py-4 text-sm text-studio-muted">
        <Loader2 className="h-4 w-4 animate-spin" />
        加载日更工作流…
      </section>
    );
  }
  if (!data) return null;

  const wordsPct =
    data.today_words_goal > 0
      ? Math.min(100, Math.round((data.today_words / data.today_words_goal) * 100))
      : 0;
  const chaptersPct =
    data.today_chapters_goal > 0
      ? Math.min(100, Math.round((data.today_chapters / data.today_chapters_goal) * 100))
      : 0;

  return (
    <section className="overflow-hidden rounded-2xl border border-studio-border bg-studio-panel shadow-sm">
      <div className="flex items-start justify-between gap-3 border-b border-studio-border/60 px-5 py-4">
        <div className="flex items-start gap-3">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-studio-accent/15">
            <Target className="h-4 w-4 text-studio-accent" />
          </div>
          <div>
            <h3 className="text-sm font-semibold text-studio-text">今日连载</h3>
            <p className="mt-0.5 text-xs text-studio-muted">
              {data.goal_met_today ? (
                <span className="inline-flex items-center gap-1 text-[rgb(var(--studio-diff-add-stat))]">
                  <CheckCircle2 className="h-3.5 w-3.5" />
                  今日目标已达成
                </span>
              ) : (
                <>目标 {data.today_chapters_goal} 章 / {data.today_words_goal.toLocaleString()} 字</>
              )}
            </p>
          </div>
        </div>
        <button
          type="button"
          onClick={() => setSettingsOpen((v) => !v)}
          className="rounded-lg p-1.5 text-studio-muted hover:bg-studio-bg hover:text-studio-text"
          title="日更目标设置"
        >
          <Settings2 className="h-4 w-4" />
        </button>
      </div>

      {settingsOpen && (
        <div className="border-b border-studio-border/60 bg-studio-bg/40 px-5 py-3">
          <div className="grid gap-3 sm:grid-cols-3">
            <label className="block text-xs text-studio-muted">
              每日章数
              <input
                type="number"
                min={1}
                value={dailyChapters}
                onChange={(e) => setDailyChapters(Number(e.target.value) || 1)}
                className="mt-1 w-full rounded-lg border border-studio-border bg-studio-panel px-2 py-1.5 text-sm"
              />
            </label>
            <label className="block text-xs text-studio-muted">
              每日字数（0=自动）
              <input
                type="number"
                min={0}
                step={500}
                value={dailyWords}
                onChange={(e) => setDailyWords(Number(e.target.value) || 0)}
                className="mt-1 w-full rounded-lg border border-studio-border bg-studio-panel px-2 py-1.5 text-sm"
              />
            </label>
            <label className="block text-xs text-studio-muted">
              存稿目标（章）
              <input
                type="number"
                min={1}
                value={bufferTarget}
                onChange={(e) => setBufferTarget(Number(e.target.value) || 7)}
                className="mt-1 w-full rounded-lg border border-studio-border bg-studio-panel px-2 py-1.5 text-sm"
              />
            </label>
          </div>
          <div className="mt-3 flex justify-end gap-2">
            <button
              type="button"
              onClick={() => setSettingsOpen(false)}
              className="rounded-lg border border-studio-border px-3 py-1.5 text-xs hover:bg-studio-bg"
            >
              取消
            </button>
            <button
              type="button"
              onClick={() => void saveSettings()}
              disabled={saving}
              className="rounded-lg bg-studio-accent px-3 py-1.5 text-xs font-medium text-studio-on-accent disabled:opacity-50"
            >
              {saving ? "保存中…" : "保存"}
            </button>
          </div>
        </div>
      )}

      <div className="grid gap-4 px-5 py-4 sm:grid-cols-3">
        <div>
          <p className="text-[10px] font-medium uppercase tracking-wide text-studio-muted">今日进度</p>
          <p className="mt-1 text-lg font-semibold tabular-nums">
            {data.today_chapters}/{data.today_chapters_goal} 章
          </p>
          <div className="mt-2 h-1.5 overflow-hidden rounded-full bg-studio-bg">
            <div
              className="h-full rounded-full bg-studio-accent transition-all"
              style={{ width: `${chaptersPct}%` }}
            />
          </div>
          <p className="mt-1 text-[11px] text-studio-muted">
            {data.today_words.toLocaleString()} / {data.today_words_goal.toLocaleString()} 字 ({wordsPct}%)
          </p>
        </div>

        <div>
          <p className="text-[10px] font-medium uppercase tracking-wide text-studio-muted">连续写作</p>
          <div className="mt-1 flex items-baseline gap-2">
            <Flame className="h-5 w-5 text-orange-400" />
            <span className="text-2xl font-semibold tabular-nums">{data.current_streak}</span>
            <span className="text-xs text-studio-muted">天</span>
          </div>
          <p className="mt-1 text-[11px] text-studio-muted">最长 {data.longest_streak} 天</p>
          <div className="mt-2 flex items-end gap-0.5">
            {data.calendar.slice(-7).map((d) => (
              <div
                key={d.date}
                title={`${d.date}: ${d.chapters} 章`}
                className={`h-6 flex-1 rounded-sm ${
                  d.goal_met
                    ? "bg-studio-accent/70"
                    : d.chapters > 0
                      ? "bg-studio-accent/25"
                      : "bg-studio-bg"
                }`}
              />
            ))}
          </div>
        </div>

        <div>
          <p className="text-[10px] font-medium uppercase tracking-wide text-studio-muted">存稿缓冲</p>
          <p className={`mt-1 text-lg font-semibold tabular-nums ${data.buffer_ok ? "text-[rgb(var(--studio-diff-add-stat))]" : ""}`}>
            {data.buffer_ready} / {data.buffer_target} 章
          </p>
          <p className="mt-1 text-[11px] text-studio-muted">
            未发布共 {data.buffer_count} 章（已审 {data.buffer_ready} 章）
          </p>
          {!data.buffer_ok && (
            <p className="mt-1 text-[11px] text-[rgb(var(--studio-warning-fg))]">
              建议再备 {data.buffer_target - data.buffer_ready} 章存稿
            </p>
          )}
        </div>
      </div>

      {data.suggestions.length > 0 && (
        <div className="border-t border-studio-border/60 px-5 py-3">
          <p className="mb-2 flex items-center gap-1.5 text-[10px] font-medium uppercase tracking-wide text-studio-muted">
            <Calendar className="h-3 w-3" />
            今日建议
          </p>
          <ul className="space-y-2">
            {data.suggestions.map((todo) => (
              <li key={todo.id}>
                <button
                  type="button"
                  onClick={() => runAction(todo)}
                  className="flex w-full items-start gap-2 rounded-lg border border-studio-border/60 bg-studio-bg/40 px-3 py-2 text-left transition hover:border-studio-border hover:bg-studio-bg"
                >
                  <div className="min-w-0 flex-1">
                    <p className="text-sm font-medium text-studio-text">{todo.label}</p>
                    {todo.detail && (
                      <p className="mt-0.5 text-xs text-studio-muted">{todo.detail}</p>
                    )}
                  </div>
                  <ChevronRight className="mt-0.5 h-4 w-4 shrink-0 text-studio-muted" />
                </button>
              </li>
            ))}
          </ul>
        </div>
      )}
    </section>
  );
}
