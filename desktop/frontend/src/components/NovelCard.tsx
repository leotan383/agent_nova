import { BookOpen, Pin } from "lucide-react";
import { NovelCard, formatRelativeTime, formatWordCount, phaseLabel } from "../lib/wails";

const phaseColors: Record<string, string> = {
  init_done: "text-studio-ai bg-studio-ai/10",
  planning: "text-studio-ai bg-studio-ai/10",
  writing: "text-studio-accent bg-studio-accent/10",
  paused: "text-studio-muted bg-studio-muted/10",
};

type Props = {
  novel: NovelCard;
  active?: boolean;
  onOpen: () => void;
  onReveal: () => void;
  onRemove: () => void;
  onTogglePin: () => void;
};

export default function NovelCardView({
  novel,
  active,
  onOpen,
  onReveal,
  onRemove,
  onTogglePin,
}: Props) {
  const initial = (novel.title || "?").charAt(0);
  const phase = phaseLabel[novel.phase] || novel.phase || "未知";

  return (
    <article
      className={`group relative flex flex-col overflow-hidden rounded-xl border bg-studio-panel transition hover:-translate-y-0.5 hover:shadow-card ${
        active
          ? "border-studio-accent/60 ring-1 ring-studio-accent/30"
          : "border-studio-border hover:border-studio-muted/40"
      } ${novel.missing ? "opacity-60" : ""}`}
    >
      <div className="relative h-28 bg-gradient-to-br from-studio-cover-from via-studio-cover-via to-studio-cover-to">
        <div className="absolute inset-0 flex items-center justify-center">
          <span className="font-serif text-4xl text-studio-accent/80">{initial}</span>
        </div>
        {novel.pinned && (
          <Pin className="absolute right-3 top-3 h-4 w-4 text-studio-accent" />
        )}
        {novel.missing && (
          <span className="absolute left-3 top-3 rounded bg-[rgb(var(--studio-danger-bg))] px-2 py-0.5 text-xs text-[rgb(var(--studio-danger-fg))]">
            路径失效
          </span>
        )}
      </div>

      <div className="flex flex-1 flex-col gap-3 p-4">
        <div>
          <h3 className="truncate text-lg font-medium">{novel.title || "未命名"}</h3>
          <p className="mt-1 text-sm text-studio-muted">
            {novel.genre || "未分类"}
            {novel.chapter_count > 0 && ` · ${novel.chapter_count} 章`}
          </p>
        </div>

        {novel.target_words != null && novel.target_words > 0 && (
          <div>
            <div className="mb-1 flex justify-between text-xs text-studio-muted">
              <span>
                {formatWordCount(novel.written_words ?? 0)} / {formatWordCount(novel.target_words)}
              </span>
              <span>{(novel.progress_percent ?? 0).toFixed(0)}%</span>
            </div>
            <div className="h-1.5 overflow-hidden rounded-full bg-studio-bg">
              <div
                className="h-full rounded-full bg-studio-accent/80"
                style={{ width: `${Math.min(100, novel.progress_percent ?? 0)}%` }}
              />
            </div>
          </div>
        )}

        <div className="flex items-center gap-2">
          <span
            className={`rounded-full px-2 py-0.5 text-xs ${
              phaseColors[novel.phase] || "text-studio-muted bg-studio-border"
            }`}
          >
            {phase}
          </span>
          <span className="text-xs text-studio-muted">
            {formatRelativeTime(novel.last_opened_at)}
          </span>
        </div>

        <div className="mt-auto flex gap-2 pt-2">
          <button
            type="button"
            onClick={onOpen}
            disabled={novel.missing}
            className="inline-flex flex-1 items-center justify-center gap-1.5 rounded-lg bg-studio-accent px-3 py-2 text-sm font-medium text-studio-on-accent transition hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-40"
          >
            <BookOpen className="h-4 w-4" />
            进入
          </button>
          <button
            type="button"
            onClick={onTogglePin}
            className="rounded-lg border border-studio-border px-3 py-2 text-xs text-studio-muted hover:border-studio-muted hover:text-studio-text"
          >
            {novel.pinned ? "取消置顶" : "置顶"}
          </button>
        </div>

        <div className="flex gap-3 text-xs text-studio-muted opacity-0 transition group-hover:opacity-100">
          <button type="button" onClick={onReveal} className="hover:text-studio-text">
            在文件夹中显示
          </button>
          <button type="button" onClick={onRemove} className="hover:text-[rgb(var(--studio-danger-fg))]">
            从书库移除
          </button>
        </div>
      </div>
    </article>
  );
}
