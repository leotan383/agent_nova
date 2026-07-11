import { useCallback, useEffect, useState } from "react";
import { BookOpen, Square } from "lucide-react";
import { BOOK_READ_EVENTS, eventsOn } from "../lib/runtime";
import { BookReadReportDTO, StartBookReadInput, app } from "../lib/wails";

type Props = {
  currentChapter: number;
  onOpenChapter?: (chapter: number) => void;
};

export default function BookReadReportPanel({ currentChapter, onOpenChapter }: Props) {
  const [focus, setFocus] = useState("all");
  const [fromChapter, setFromChapter] = useState(1);
  const [toChapter, setToChapter] = useState(Math.max(1, currentChapter));
  const [running, setRunning] = useState(false);
  const [jobId, setJobId] = useState("");
  const [message, setMessage] = useState("");
  const [report, setReport] = useState<BookReadReportDTO | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    if (currentChapter > 0) {
      setToChapter((prev) => (prev > currentChapter ? currentChapter : prev));
    }
  }, [currentChapter]);

  const restoreJob = useCallback(async () => {
    try {
      const active = await app().GetActiveBookReadJob();
      if (!active.id) return;
      setJobId(active.id);
      setRunning(active.status === "running" || active.status === "pending");
      if (active.status === "done") {
        const rep = await app().GetBookReadReport(active.id);
        setReport(rep);
        setRunning(false);
      }
    } catch {
      /* ignore */
    }
  }, []);

  useEffect(() => {
    void restoreJob();
  }, [restoreJob]);

  useEffect(() => {
    const offs = [
      eventsOn(BOOK_READ_EVENTS.status, (p) => {
        if (p.job_id !== jobId && jobId) return;
        setMessage(String(p.message ?? ""));
        if (p.status === "running" || p.status === "pending") setRunning(true);
        if (p.status === "failed" || p.status === "cancelled") setRunning(false);
      }),
      eventsOn(BOOK_READ_EVENTS.done, (p) => {
        if (p.job_id !== jobId && jobId) return;
        setRunning(false);
        try {
          setReport(JSON.parse(String(p.report ?? "{}")) as BookReadReportDTO);
        } catch {
          setError("报告解析失败");
        }
      }),
      eventsOn(BOOK_READ_EVENTS.error, (p) => {
        if (p.job_id !== jobId && jobId) return;
        setRunning(false);
        setError(String(p.error ?? "通读失败"));
      }),
    ];
    return () => offs.forEach((o) => o());
  }, [jobId]);

  const start = useCallback(async () => {
    setError("");
    setReport(null);
    setRunning(true);
    try {
      const from = Math.max(1, fromChapter);
      const to = Math.min(Math.max(from, toChapter), Math.max(1, currentChapter));
      const input: StartBookReadInput = { from_chapter: from, to_chapter: to, focus };
      const job = await app().StartBookReadReport(input);
      setJobId(job.id);
    } catch (e) {
      setRunning(false);
      setError(e instanceof Error ? e.message : String(e));
    }
  }, [currentChapter, focus, fromChapter, toChapter]);

  const cancel = async () => {
    if (!jobId) return;
    try {
      await app().CancelBookReadReport(jobId);
      setRunning(false);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const severityClass: Record<string, string> = {
    critical: "border-[rgb(var(--studio-danger-border))] bg-[rgb(var(--studio-danger-bg))]",
    warn: "border-amber-500/30 bg-amber-500/10",
    info: "border-studio-border bg-studio-bg/40",
  };

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-end gap-2">
        <label className="text-xs text-studio-muted">
          章节范围
          <div className="mt-1 flex items-center gap-1">
            <input
              type="number"
              min={1}
              max={currentChapter}
              value={fromChapter}
              disabled={running}
              onChange={(e) => setFromChapter(parseInt(e.target.value, 10) || 1)}
              className="w-16 rounded-md border border-studio-border bg-studio-bg px-2 py-1 text-sm"
            />
            <span>—</span>
            <input
              type="number"
              min={1}
              max={currentChapter}
              value={toChapter}
              disabled={running}
              onChange={(e) => setToChapter(parseInt(e.target.value, 10) || 1)}
              className="w-16 rounded-md border border-studio-border bg-studio-bg px-2 py-1 text-sm"
            />
          </div>
        </label>
        <label className="text-xs text-studio-muted">
          关注维度
          <select
            value={focus}
            onChange={(e) => setFocus(e.target.value)}
            disabled={running}
            className="ml-0 mt-1 block rounded-md border border-studio-border bg-studio-bg px-2 py-1 text-sm"
          >
            <option value="all">全部</option>
            <option value="pace">节奏</option>
            <option value="repeat">重复</option>
            <option value="foreshadow">伏笔</option>
          </select>
        </label>
        {running ? (
          <button
            type="button"
            onClick={() => void cancel()}
            className="inline-flex items-center gap-1.5 rounded-lg border border-studio-border px-3 py-1.5 text-sm hover:bg-studio-bg"
          >
            <Square className="h-4 w-4" />
            取消
          </button>
        ) : (
          <button
            type="button"
            disabled={currentChapter <= 0}
            onClick={() => void start()}
            className="inline-flex items-center gap-1.5 rounded-lg bg-studio-accent px-3 py-1.5 text-sm font-medium text-studio-on-accent disabled:opacity-40"
          >
            <BookOpen className="h-4 w-4" />
            生成通读报告
          </button>
        )}
      </div>
      {running && message && <p className="text-xs text-studio-muted">{message}</p>}
      {error && <p className="text-xs text-[rgb(var(--studio-danger-fg))]">{error}</p>}
      {report && (
        <div className="space-y-3">
          <p className="rounded-lg border border-studio-border bg-studio-bg/50 px-3 py-2 text-sm">{report.summary}</p>
          <ul className="space-y-2">
            {report.items?.map((it, i) => (
              <li
                key={`${it.chapter}-${i}`}
                className={`rounded-lg border px-3 py-2 text-xs ${severityClass[it.severity] ?? severityClass.info}`}
              >
                <div className="flex items-center gap-2">
                  <span className="font-medium">
                    第 {it.chapter} 章 · {it.title}
                  </span>
                  <span className="text-studio-muted">[{it.category}]</span>
                  {onOpenChapter && (
                    <button
                      type="button"
                      onClick={() => onOpenChapter(it.chapter)}
                      className="ml-auto text-studio-accent hover:underline"
                    >
                      打开
                    </button>
                  )}
                </div>
                <p className="mt-1 text-studio-text">{it.excerpt}</p>
                <p className="mt-1 text-studio-muted">建议：{it.suggestion}</p>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
