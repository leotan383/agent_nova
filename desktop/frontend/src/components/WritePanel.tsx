import { useCallback, useEffect, useRef, useState } from "react";
import { Loader2, Square, Wand2 } from "lucide-react";
import { WRITE_EVENTS, eventsOn } from "../lib/runtime";
import { StatusReport, WriteReportDTO, app } from "../lib/wails";
import WriteContextPanel from "./WriteContextPanel";

const stepLabels: Record<string, string> = {
  gate: "写前检查",
  context: "组装上下文",
  taskbook: "任务书",
  draft: "起草正文",
  review: "审查润色",
  summary: "生成摘要",
  extract: "沉淀记忆",
  done: "完成",
};

type Props = {
  status: StatusReport | null;
  onComplete: () => void;
};

export default function WritePanel({ status, onComplete }: Props) {
  const [chapter, setChapter] = useState(1);
  const [volume, setVolume] = useState(1);
  const [resume, setResume] = useState(false);
  const [jobId, setJobId] = useState("");
  const [jobStatus, setJobStatus] = useState("");
  const [step, setStep] = useState("");
  const [stepMessage, setStepMessage] = useState("");
  const [streamText, setStreamText] = useState("");
  const [report, setReport] = useState<WriteReportDTO | null>(null);
  const [error, setError] = useState("");
  const [hasKey, setHasKey] = useState(true);
  const [running, setRunning] = useState(false);
  const scrollRef = useRef<HTMLPreElement>(null);
  const jobIdRef = useRef("");

  useEffect(() => {
    if (status) {
      setChapter(Math.max(1, status.current_chapter + 1));
      setVolume(Math.max(1, status.current_volume || 1));
    }
  }, [status]);

  useEffect(() => {
    app()
      .HasAPIKey()
      .then(setHasKey)
      .catch(() => setHasKey(false));
  }, []);

  useEffect(() => {
    const match = (id?: string) => !jobIdRef.current || id === jobIdRef.current;
    const unsubs = [
      eventsOn(WRITE_EVENTS.step, (p) => {
        if (!match(p.job_id)) return;
        setStep(p.step || "");
        setStepMessage(p.message || "");
      }),
      eventsOn(WRITE_EVENTS.delta, (p) => {
        if (!match(p.job_id)) return;
        setStreamText((t) => t + (p.delta || ""));
      }),
      eventsOn(WRITE_EVENTS.status, (p) => {
        if (!match(p.job_id)) return;
        setJobStatus(p.status || "");
        if (p.status === "running") setRunning(true);
        if (p.status === "done" || p.status === "failed" || p.status === "cancelled") {
          setRunning(false);
        }
      }),
      eventsOn(WRITE_EVENTS.error, (p) => {
        if (!match(p.job_id)) return;
        setError(p.error || "写章失败");
        setRunning(false);
      }),
      eventsOn(WRITE_EVENTS.done, (p) => {
        if (!match(p.job_id)) return;
        if (p.report) {
          try {
            setReport(JSON.parse(p.report) as WriteReportDTO);
          } catch {
            /* ignore */
          }
        }
        setRunning(false);
        onComplete();
      }),
    ];
    return () => unsubs.forEach((u) => u());
  }, [onComplete]);

  useEffect(() => {
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight });
  }, [streamText]);

  const startWrite = useCallback(async () => {
    setError("");
    setReport(null);
    setStreamText("");
    setStep("");
    setStepMessage("");
    setJobStatus("");
    try {
      const job = await app().StartWriteChapter({
        chapter,
        volume,
        resume,
      });
      jobIdRef.current = job.id;
      setJobId(job.id);
      setJobStatus(job.status);
      setRunning(true);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, [chapter, volume, resume]);

  const cancelWrite = async () => {
    if (!jobId) return;
    try {
      await app().CancelWriteChapter(jobId);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  return (
    <div className="flex h-full min-h-0 flex-col gap-5 overflow-hidden">
      <WriteContextPanel chapter={chapter} volume={volume} />

      <div className="flex flex-wrap items-end gap-4 rounded-xl border border-studio-border bg-studio-panel p-5">
        <div>
          <label className="mb-1 block text-xs text-studio-muted">章号</label>
          <input
            type="number"
            min={1}
            value={chapter}
            onChange={(e) => setChapter(Number(e.target.value))}
            disabled={running}
            className="w-24 rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none focus:border-studio-accent disabled:opacity-50"
          />
        </div>
        <div>
          <label className="mb-1 block text-xs text-studio-muted">卷号</label>
          <input
            type="number"
            min={1}
            value={volume}
            onChange={(e) => setVolume(Number(e.target.value))}
            disabled={running}
            className="w-24 rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none focus:border-studio-accent disabled:opacity-50"
          />
        </div>
        <label className="flex items-center gap-2 pb-2 text-sm text-studio-muted">
          <input
            type="checkbox"
            checked={resume}
            onChange={(e) => setResume(e.target.checked)}
            disabled={running}
            className="rounded border-studio-border"
          />
          断点续写
        </label>
        <div className="ml-auto flex gap-2">
          {running ? (
            <button
              type="button"
              onClick={cancelWrite}
              className="inline-flex items-center gap-2 rounded-lg border border-[rgb(var(--studio-danger-border))] px-4 py-2 text-sm text-[rgb(var(--studio-danger-fg))] hover:bg-[rgb(var(--studio-danger-bg))]"
            >
              <Square className="h-4 w-4" />
              取消
            </button>
          ) : (
            <button
              type="button"
              onClick={startWrite}
              disabled={!hasKey}
              className="inline-flex items-center gap-2 rounded-lg bg-studio-accent px-5 py-2 text-sm font-medium text-studio-on-accent hover:brightness-110 disabled:opacity-40"
            >
              <Wand2 className="h-4 w-4" />
              开始写章
            </button>
          )}
        </div>
      </div>

      {!hasKey && (
        <div className="studio-alert-warning">
          未配置 API Key。请在终端运行 <code className="text-studio-accent">nova config set api_key ...</code>
        </div>
      )}

      {error && (
        <div className="studio-alert-error">
          {error}
        </div>
      )}

      {(running || step) && (
        <div className="flex items-center gap-3 text-sm text-studio-muted">
          {running && <Loader2 className="h-4 w-4 animate-spin text-studio-accent" />}
          <span>
            {stepLabels[step] || step || "准备中"}
            {stepMessage ? ` · ${stepMessage}` : ""}
          </span>
          {jobStatus && (
            <span className="rounded-full bg-studio-border px-2 py-0.5 text-xs">{jobStatus}</span>
          )}
        </div>
      )}

      <div className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl border border-studio-border bg-studio-paper">
        <pre
          ref={scrollRef}
          className="min-h-0 flex-1 overflow-y-auto whitespace-pre-wrap p-6 font-serif text-base leading-relaxed text-studio-ink"
        >
          {streamText || (running ? "等待流式输出…" : "点击「开始写章」运行完整写章流水线（起草 → 审查 → 摘要 → 记忆）")}
        </pre>
      </div>

      {report && (
        <div
          className={`rounded-xl border p-5 ${
            report.status === "completed"
              ? "border-[rgb(var(--studio-diff-add-stat)/0.3)] bg-[rgb(var(--studio-diff-add-bg))]"
              : report.status === "needs_action"
                ? "border-[rgb(var(--studio-warning-border))] bg-[rgb(var(--studio-warning-bg))]"
                : "border-studio-border bg-studio-panel"
          }`}
        >
          <h3 className="font-medium">{report.stage}</h3>
          <p className="mt-1 text-sm text-studio-muted">{report.summary}</p>
          {report.issues && report.issues.length > 0 && (
            <ul className="mt-3 list-inside list-disc text-sm text-[rgb(var(--studio-warning-fg))]">
              {report.issues.map((i) => (
                <li key={i}>{i}</li>
              ))}
            </ul>
          )}
          {report.next_steps && report.next_steps.length > 0 && (
            <ul className="mt-2 list-inside list-disc text-sm text-studio-muted">
              {report.next_steps.map((n) => (
                <li key={n}>{n}</li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}
