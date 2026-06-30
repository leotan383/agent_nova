import { ArrowRight, BookOpen, ClipboardCheck, PenLine } from "lucide-react";
import { WriteReportDTO } from "../lib/wails";

type Props = {
  chapter: number;
  report: WriteReportDTO;
  onReview: () => void;
  onReadChapter: () => void;
  onWriteNext: () => void;
};

export default function WriteCompletionBar({
  chapter,
  report,
  onReview,
  onReadChapter,
  onWriteNext,
}: Props) {
  const ok = report.status === "completed";

  return (
    <div
      className={`shrink-0 rounded-xl border p-4 ${
        ok
          ? "border-[rgb(var(--studio-diff-add-stat)/0.3)] bg-[rgb(var(--studio-diff-add-bg))]"
          : "border-[rgb(var(--studio-warning-border))] bg-[rgb(var(--studio-warning-bg))]"
      }`}
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-sm font-medium text-studio-text">
            {ok ? `第 ${chapter} 章写作完成` : report.stage || "写章结束"}
          </p>
          {report.summary && (
            <p className="mt-1 text-xs text-studio-muted">{report.summary}</p>
          )}
          {report.issues && report.issues.length > 0 && (
            <ul className="mt-2 space-y-0.5 text-xs text-[rgb(var(--studio-warning-fg))]">
              {report.issues.map((i) => (
                <li key={i}>· {i}</li>
              ))}
            </ul>
          )}
        </div>
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            onClick={onReview}
            className="inline-flex items-center gap-1.5 rounded-lg border border-studio-border bg-studio-panel px-3 py-1.5 text-xs hover:bg-studio-bg"
          >
            <ClipboardCheck className="h-3.5 w-3.5" />
            审查本章
          </button>
          <button
            type="button"
            onClick={onReadChapter}
            className="inline-flex items-center gap-1.5 rounded-lg border border-studio-border bg-studio-panel px-3 py-1.5 text-xs hover:bg-studio-bg"
          >
            <BookOpen className="h-3.5 w-3.5" />
            阅读正文
          </button>
          <button
            type="button"
            onClick={onWriteNext}
            className="inline-flex items-center gap-1.5 rounded-lg bg-studio-accent px-3 py-1.5 text-xs font-medium text-studio-on-accent hover:brightness-110"
          >
            <PenLine className="h-3.5 w-3.5" />
            写第 {chapter + 1} 章
            <ArrowRight className="h-3 w-3" />
          </button>
        </div>
      </div>
    </div>
  );
}
