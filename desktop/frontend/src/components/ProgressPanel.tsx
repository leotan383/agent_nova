import { StatusReport, formatWordCount } from "../lib/wails";

type Props = {
  status: StatusReport;
};

export default function ProgressPanel({ status }: Props) {
  const pct = Math.min(100, Math.max(0, status.progress_percent ?? 0));
  const hasTarget = (status.target_words ?? 0) > 0;

  return (
    <div className="rounded-xl border border-studio-border bg-studio-panel p-5 md:col-span-2 xl:col-span-3">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h3 className="text-sm font-medium text-studio-muted">创作进度</h3>
          <p className="mt-1 text-2xl font-semibold text-studio-text">
            {formatWordCount(status.written_words ?? 0)}
            <span className="mx-2 text-base font-normal text-studio-muted">/</span>
            <span className="text-lg font-medium text-studio-accent">
              {formatWordCount(status.target_words ?? 0)}
            </span>
          </p>
        </div>
        <div className="text-right text-sm text-studio-muted">
          <p className="text-2xl font-semibold tabular-nums text-studio-accent">
            {pct.toFixed(1)}%
          </p>
          {status.remaining_chapters > 0 && (
            <p className="mt-1">预计还需 {status.remaining_chapters} 章</p>
          )}
        </div>
      </div>

      {hasTarget && (
        <div className="mt-4 h-2 overflow-hidden rounded-full bg-studio-bg">
          <div
            className="h-full rounded-full bg-gradient-to-r from-studio-accent/80 to-studio-accent transition-all duration-500"
            style={{ width: `${pct}%` }}
          />
        </div>
      )}

      <div className="mt-4 flex flex-wrap gap-x-6 gap-y-1 text-xs text-studio-muted">
        <span>已写 {status.chapter_count} 章</span>
        {status.estimated_total_chapters > 0 && (
          <span>目标约 {status.estimated_total_chapters} 章</span>
        )}
        {status.chapter_words_goal > 0 && (
          <span>每章目标 {formatWordCount(status.chapter_words_goal)}</span>
        )}
        {status.avg_words_per_chapter > 0 && (
          <span>实际章均 {formatWordCount(status.avg_words_per_chapter)}</span>
        )}
        {status.style && <span>风格 {status.style}</span>}
        {status.remaining_words > 0 && (
          <span>还差 {formatWordCount(status.remaining_words)}</span>
        )}
      </div>
    </div>
  );
}
