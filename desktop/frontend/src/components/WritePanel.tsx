import { useCallback, useEffect, useRef, useState } from "react";
import {
  ChevronDown,
  Loader2,
  PanelLeftClose,
  PanelLeftOpen,
  Square,
  Wand2,
} from "lucide-react";
import { WRITE_EVENTS, eventsOn } from "../lib/runtime";
import { StatusReport, WriteReportDTO, WriteJobStateDTO, app } from "../lib/wails";
import WriteCompletionBar from "./WriteCompletionBar";
import WriteContextPanel from "./WriteContextPanel";
import WriteGatePanel from "./WriteGatePanel";
import WriteStepper from "./WriteStepper";

const jobStatusLabels: Record<string, string> = {
  pending: "排队中",
  running: "进行中",
  done: "已完成",
  failed: "失败",
  cancelled: "已取消",
};

const stepLabels: Record<string, string> = {
  gate: "写前检查",
  context: "组装上下文",
  taskbook: "生成任务书",
  draft: "起草正文",
  review: "审查润色",
  summary: "生成摘要",
  extract: "沉淀记忆",
  done: "完成",
};

type Props = {
  status: StatusReport | null;
  embedded?: boolean;
  onComplete: () => void;
  onGoToPlanning: (volume?: number) => void;
  onReviewChapter: (chapter: number) => void;
  onReadChapter: (chapter: number) => void;
  onRebuildIndex: () => Promise<void>;
};

export default function WritePanel({
  status,
  embedded = false,
  onComplete,
  onGoToPlanning,
  onReviewChapter,
  onReadChapter,
  onRebuildIndex,
}: Props) {
  const nextChapter = Math.max(1, (status?.current_chapter ?? 0) + 1);
  const volume = Math.max(1, status?.current_volume ?? 1);

  const [writingChapter, setWritingChapter] = useState<number | null>(null);
  const chapter = writingChapter ?? nextChapter;
  const [resume, setResume] = useState(false);
  const [jobId, setJobId] = useState("");
  const [jobStatus, setJobStatus] = useState("");
  const [step, setStep] = useState("");
  const [stepMessage, setStepMessage] = useState("");
  const [streamText, setStreamText] = useState("");
  const [report, setReport] = useState<WriteReportDTO | null>(null);
  const [error, setError] = useState("");
  const [hasKey, setHasKey] = useState(true);
  const [gateOK, setGateOK] = useState(false);
  const [running, setRunning] = useState(false);
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [optionsOpen, setOptionsOpen] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);
  const jobIdRef = useRef("");

  useEffect(() => {
    app()
      .HasAPIKey()
      .then(setHasKey)
      .catch(() => setHasKey(false));
  }, []);

  useEffect(() => {
    void (async () => {
      try {
        const { active, job, state } = await app().GetActiveWriteJob();
        if (!active || !job.id) return;
        applyJobState(job.id, job.chapter, job.status, state);
      } catch {
        /* ignore */
      }
    })();
  }, []);

  const applyJobState = (
    id: string,
    chapterNum: number,
    status: string,
    state?: WriteJobStateDTO,
  ) => {
    jobIdRef.current = id;
    setJobId(id);
    setWritingChapter(chapterNum);
    setJobStatus(status);
    const isRunning = status === "pending" || status === "running";
    setRunning(isRunning);
    if (state?.stream_text) setStreamText(state.stream_text);
    if (state?.step) setStep(state.step);
    if (state?.step_message) setStepMessage(state.step_message);
    if (isRunning) setSidebarOpen(false);
  };

  // 切 Tab 回来后从后端拉全量流式缓冲
  useEffect(() => {
    if (!running || !jobId) return;
    const sync = () => {
      app()
        .GetWriteJobState(jobId)
        .then((state) => {
          if (state.stream_text) setStreamText(state.stream_text);
          if (state.step) setStep(state.step);
          if (state.step_message) setStepMessage(state.step_message);
        })
        .catch(() => {});
    };
    sync();
    const timer = window.setInterval(sync, 3000);
    return () => window.clearInterval(timer);
  }, [running, jobId]);

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
        if (p.status === "running") {
          setRunning(true);
          setSidebarOpen(false);
        }
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
        setStep("done");
        onComplete();
      }),
    ];
    return () => unsubs.forEach((u) => u());
  }, [onComplete]);

  useEffect(() => {
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight, behavior: "smooth" });
  }, [streamText]);

  const startWrite = useCallback(async () => {
    setError("");
    setReport(null);
    setStreamText("");
    setStep("");
    setStepMessage("");
    setJobStatus("");
    try {
      setWritingChapter(nextChapter);
      const job = await app().StartWriteChapter({ chapter: nextChapter, volume, resume });
      jobIdRef.current = job.id;
      setJobId(job.id);
      setJobStatus(job.status);
      setRunning(true);
      setSidebarOpen(false);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, [nextChapter, volume, resume]);

  const cancelWrite = async () => {
    if (!jobId) return;
    try {
      await app().CancelWriteChapter(jobId);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const handleGateFix = (key: string) => {
    switch (key) {
      case "volume_outline":
      case "chapter_outline":
        onGoToPlanning(volume);
        break;
      case "prev_summary":
        if (chapter > 1) onReadChapter(chapter - 1);
        break;
      case "index":
        void onRebuildIndex();
        break;
      default:
        break;
    }
  };

  const wordCount = streamText.replace(/\s/g, "").length;
  const alertMessage =
    !hasKey
      ? "未配置 API Key，请先在右上角「设置」中填写。"
      : !gateOK && !running
        ? "写前检查未通过，请展开侧栏处理阻塞项。"
        : error;

  return (
    <div className="flex h-full min-h-0 flex-col overflow-hidden">
      <header
        className={`shrink-0 border-b border-studio-border px-5 py-4 ${
          embedded ? "bg-studio-panel" : "bg-studio-panel/80 backdrop-blur-sm"
        }`}
      >
        <div className="flex flex-wrap items-center gap-3">
          <div className="min-w-0">
            <p className="text-[10px] font-medium uppercase tracking-wide text-studio-muted">写作任务</p>
            <div className="mt-0.5 flex flex-wrap items-baseline gap-x-2 gap-y-1">
              <span className="text-lg font-semibold text-studio-text">
                第 {volume} 卷 · 第 {chapter} 章
              </span>
              {!running && status && status.current_chapter > 0 && (
                <span className="text-xs text-studio-muted">
                  接在第 {status.current_chapter} 章之后
                </span>
              )}
            </div>
          </div>

          <div className="hidden h-8 w-px bg-studio-border sm:block" />

          {running ? (
            <span className="inline-flex items-center gap-1.5 rounded-full bg-studio-accent/15 px-2.5 py-1 text-xs font-medium text-studio-accent">
              <Loader2 className="h-3 w-3 animate-spin" />
              {stepLabels[step] || "写作中"}
            </span>
          ) : report ? (
            <span className="inline-flex items-center gap-1 rounded-full bg-[rgb(var(--studio-diff-add-bg))] px-2.5 py-1 text-xs text-[rgb(var(--studio-diff-add-stat))]">
              已完成
            </span>
          ) : gateOK ? (
            <span className="inline-flex items-center gap-1 rounded-full bg-[rgb(var(--studio-diff-add-bg))] px-2.5 py-1 text-xs text-[rgb(var(--studio-diff-add-stat))]">
              可开写
            </span>
          ) : (
            <span className="inline-flex items-center gap-1 rounded-full bg-[rgb(var(--studio-warning-bg))] px-2.5 py-1 text-xs text-[rgb(var(--studio-warning-fg))]">
              检查未通过
            </span>
          )}

          <div className="ml-auto flex flex-wrap items-center gap-2">
            <button
              type="button"
              onClick={() => setSidebarOpen((v) => !v)}
              className="inline-flex items-center gap-1 rounded-lg border border-studio-border px-2.5 py-2 text-xs text-studio-muted hover:bg-studio-bg hover:text-studio-text"
              title={sidebarOpen ? "收起准备面板" : "展开准备面板"}
            >
              {sidebarOpen ? (
                <PanelLeftClose className="h-3.5 w-3.5" />
              ) : (
                <PanelLeftOpen className="h-3.5 w-3.5" />
              )}
              <span className="hidden sm:inline">{sidebarOpen ? "收起" : "准备"}</span>
            </button>

            <div className="relative">
              <button
                type="button"
                onClick={() => setOptionsOpen((v) => !v)}
                disabled={running}
                className="inline-flex items-center gap-1 rounded-lg border border-studio-border px-2.5 py-2 text-xs text-studio-muted hover:bg-studio-bg disabled:opacity-50"
              >
                选项
                <ChevronDown className="h-3 w-3" />
              </button>
              {optionsOpen && (
                <label className="absolute right-0 top-full z-10 mt-1 flex items-center gap-2 whitespace-nowrap rounded-lg border border-studio-border bg-studio-panel px-3 py-2 text-xs shadow-card">
                  <input
                    type="checkbox"
                    checked={resume}
                    onChange={(e) => setResume(e.target.checked)}
                    className="rounded border-studio-border"
                  />
                  断点续写
                </label>
              )}
            </div>

            {running ? (
              <button
                type="button"
                onClick={cancelWrite}
                className="inline-flex items-center gap-2 rounded-xl border border-[rgb(var(--studio-danger-border))] px-4 py-2 text-sm text-[rgb(var(--studio-danger-fg))] hover:bg-[rgb(var(--studio-danger-bg))]"
              >
                <Square className="h-4 w-4" />
                取消
              </button>
            ) : (
              <button
                type="button"
                onClick={startWrite}
                disabled={!hasKey || !gateOK}
                className="inline-flex items-center gap-2 rounded-xl bg-studio-accent px-5 py-2 text-sm font-medium text-studio-on-accent hover:brightness-110 disabled:opacity-40"
              >
                <Wand2 className="h-4 w-4" />
                开始写章
              </button>
            )}
          </div>
        </div>
      </header>

      {alertMessage && (
        <div
          className={`shrink-0 px-5 py-2 text-sm ${
            error ? "studio-alert-error-compact" : "studio-alert-warning-compact"
          } mx-5 mt-3 rounded-lg`}
        >
          {alertMessage}
        </div>
      )}

      <div className={`flex min-h-0 flex-1 gap-4 overflow-hidden ${embedded ? "p-3" : "p-4"}`}>
        {sidebarOpen && (
          <aside className="flex w-72 shrink-0 flex-col gap-3 overflow-y-auto lg:w-80">
            <WriteGatePanel
              chapter={chapter}
              volume={volume}
              compact
              onFix={handleGateFix}
              onReadyChange={setGateOK}
            />
            <WriteContextPanel chapter={chapter} volume={volume} defaultCollapsed />
          </aside>
        )}

        <main className="flex min-h-0 min-w-0 flex-1 flex-col gap-3 overflow-hidden">
          <WriteStepper currentStep={step} running={running || !!step} />

          {(running || step) && (
            <div className="flex shrink-0 flex-col gap-1">
              <div className="flex items-center gap-2 text-xs text-studio-muted">
                {running && <Loader2 className="h-3.5 w-3.5 animate-spin text-studio-accent" />}
                <span>
                  {stepLabels[step] || step || "准备中"}
                  {stepMessage ? ` · ${stepMessage}` : ""}
                </span>
                {jobStatus && !running && (
                  <span className="rounded-full bg-studio-border px-2 py-0.5 text-[10px]">
                    {jobStatusLabels[jobStatus] || jobStatus}
                  </span>
                )}
              </div>
              {running && !streamText && step !== "draft" && (
                <p className="text-[11px] text-studio-muted/80">
                  任务进行中，请稍候…
                </p>
              )}
            </div>
          )}

          <div className="relative flex min-h-0 flex-1 flex-col overflow-hidden rounded-2xl border border-studio-border bg-studio-paper shadow-inner">
            <div
              ref={scrollRef}
              className="min-h-0 flex-1 overflow-y-auto px-6 py-8 sm:px-10"
            >
              {streamText ? (
                <div className="mx-auto max-w-3xl">
                  <p className="whitespace-pre-wrap font-serif text-base leading-[1.85] text-studio-ink sm:text-[17px]">
                    {streamText}
                  </p>
                  {running && (
                    <span className="ml-0.5 inline-block h-4 w-0.5 animate-pulse bg-studio-accent align-middle" />
                  )}
                </div>
              ) : (
                <div className="flex h-full min-h-[200px] flex-col items-center justify-center px-6 text-center">
                  <Wand2 className="mb-3 h-8 w-8 text-studio-muted/30" />
                  <p className="text-sm font-medium text-studio-muted">创作稿纸</p>
                  <p className="mt-2 max-w-sm text-xs leading-relaxed text-studio-muted/80">
                    点击「开始写章」后，AI 将依次完成检查、起草、润色、摘要与记忆沉淀。正文会在此实时显示。
                  </p>
                </div>
              )}
            </div>

            {running && streamText && (
              <div className="shrink-0 border-t border-studio-border/60 bg-studio-paper/90 px-4 py-2 text-xs text-studio-muted backdrop-blur-sm">
                正在{stepLabels[step] || "写作"}… · 已输出约 {wordCount.toLocaleString()} 字
              </div>
            )}
          </div>

          {report && !running && (
            <WriteCompletionBar
              chapter={chapter}
              report={report}
              onReview={() => onReviewChapter(chapter)}
              onReadChapter={() => onReadChapter(chapter)}
              onWriteNext={() => {
                setReport(null);
                setStreamText("");
                setStep("");
                setWritingChapter(null);
                onComplete();
              }}
            />
          )}
        </main>
      </div>
    </div>
  );
}
