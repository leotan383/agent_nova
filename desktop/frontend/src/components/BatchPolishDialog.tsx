import { useEffect, useState } from "react";
import { Check, Loader2, Wand2 } from "lucide-react";
import { POLISH_EVENTS, eventsOn } from "../lib/runtime";
import {
  BatchPolishReportDTO,
  DiffResultDTO,
  StartBatchPolishInput,
  app,
} from "../lib/wails";
import ChapterDiffView from "./ChapterDiffView";

type Props = {
  chapters: number[];
  onApplied?: () => void;
};

export default function BatchPolishDialog({ chapters, onApplied }: Props) {
  const [rule, setRule] = useState("person");
  const [running, setRunning] = useState(false);
  const [jobId, setJobId] = useState("");
  const [progress, setProgress] = useState("");
  const [report, setReport] = useState<BatchPolishReportDTO | null>(null);
  const [error, setError] = useState("");
  const [previewIdx, setPreviewIdx] = useState(0);
  const [diff, setDiff] = useState<DiffResultDTO | null>(null);
  const [applying, setApplying] = useState(false);

  useEffect(() => {
    const offs = [
      eventsOn(POLISH_EVENTS.progress, (p) => {
        if (jobId && p.job_id !== jobId) return;
        setProgress(`正在处理第 ${p.chapter} 章…`);
      }),
      eventsOn(POLISH_EVENTS.done, (p) => {
        if (jobId && p.job_id !== jobId) return;
        setRunning(false);
        try {
          setReport(JSON.parse(String(p.report ?? "{}")) as BatchPolishReportDTO);
        } catch {
          setError("结果解析失败");
        }
      }),
      eventsOn(POLISH_EVENTS.error, (p) => {
        if (jobId && p.job_id !== jobId) return;
        setRunning(false);
        setError(String(p.error ?? "润色失败"));
      }),
    ];
    return () => offs.forEach((o) => o());
  }, [jobId]);

  useEffect(() => {
    if (!report?.results?.[previewIdx]) {
      setDiff(null);
      return;
    }
    const r = report.results[previewIdx];
    if (r.error || !r.polished) {
      setDiff(null);
      return;
    }
    void app()
      .PreviewBatchPolishDiff(r.chapter, r.original, r.polished)
      .then(setDiff)
      .catch(() => setDiff(null));
  }, [report, previewIdx]);

  const start = async () => {
    if (chapters.length === 0) {
      setError("请先勾选要润色的章节");
      return;
    }
    setError("");
    setReport(null);
    setRunning(true);
    try {
      const input: StartBatchPolishInput = { chapters, rule };
      const job = await app().StartBatchPolish(input);
      setJobId(job.id);
    } catch (e) {
      setRunning(false);
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const applyCurrent = async () => {
    const r = report?.results?.[previewIdx];
    if (!r?.polished) return;
    setApplying(true);
    try {
      await app().ApplyBatchPolishChapter(r.chapter, r.polished);
      onApplied?.();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setApplying(false);
    }
  };

  const results = report?.results ?? [];

  return (
    <div className="space-y-3 rounded-lg border border-studio-border bg-studio-bg/30 p-3">
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-xs text-studio-muted">已选 {chapters.length} 章</span>
        <select
          value={rule}
          onChange={(e) => setRule(e.target.value)}
          className="rounded-md border border-studio-border bg-studio-bg px-2 py-1 text-sm"
        >
          <option value="person">统一人称</option>
          <option value="naming">统一称谓</option>
          <option value="tone">统一语气</option>
        </select>
        <button
          type="button"
          disabled={running || chapters.length === 0}
          onClick={() => void start()}
          className="inline-flex items-center gap-1.5 rounded-lg border border-studio-accent/40 bg-studio-accent/10 px-3 py-1.5 text-sm text-studio-accent disabled:opacity-40"
        >
          {running ? <Loader2 className="h-4 w-4 animate-spin" /> : <Wand2 className="h-4 w-4" />}
          批量润色
        </button>
      </div>
      {progress && running && <p className="text-xs text-studio-muted">{progress}</p>}
      {error && <p className="text-xs text-[rgb(var(--studio-danger-fg))]">{error}</p>}
      {results.length > 0 && (
        <div className="flex min-h-0 flex-col gap-2">
          <div className="flex flex-wrap gap-1">
            {results.map((r, i) => (
              <button
                key={r.chapter}
                type="button"
                onClick={() => setPreviewIdx(i)}
                className={`rounded-md px-2 py-1 text-xs ${
                  previewIdx === i ? "bg-studio-accent/20 text-studio-accent" : "bg-studio-panel text-studio-muted"
                }`}
              >
                第{r.chapter}章{r.error ? "!" : ""}
              </button>
            ))}
          </div>
          {results[previewIdx]?.error ? (
            <p className="text-xs text-[rgb(var(--studio-danger-fg))]">{results[previewIdx].error}</p>
          ) : diff ? (
            <>
              <ChapterDiffView diff={diff} maxHeight="max-h-48" />
              <button
                type="button"
                disabled={applying}
                onClick={() => void applyCurrent()}
                className="inline-flex items-center justify-center gap-1 rounded-lg bg-studio-accent px-3 py-2 text-sm text-studio-on-accent disabled:opacity-40"
              >
                {applying ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />}
                应用此章润色
              </button>
            </>
          ) : null}
        </div>
      )}
    </div>
  );
}
