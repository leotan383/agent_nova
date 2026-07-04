import { useCallback, useEffect, useRef, useState } from "react";
import { Bot, ClipboardCheck, ChevronRight, Loader2, Square } from "lucide-react";
import { AI_DETECT_EVENTS, REVIEW_EVENTS, eventsOn } from "../lib/runtime";
import {
  mergeAIDetectMetrics,
  parseAIDetectMetricsFromText,
  AIDetectMetrics,
  aiScoreColor,
  riskLevelLabel,
} from "../lib/aiDetectMetrics";
import { normalizeChapterBodyForDisplay } from "../lib/chapterBody";
import { mergeReviewMetrics, parseReviewMetricsFromText, ReviewMetrics } from "../lib/reviewMetrics";
import { AIDetectReportDTO, ChapterDocDTO, ReviewReportDTO, app } from "../lib/wails";
import { confirmUnsavedLeave } from "../lib/unsavedGuard";
import AIDetectSummaryPanel from "./AIDetectSummaryPanel";
import MarkdownEditor from "./MarkdownEditor";
import ReportSubTabBar, { ReportSubview } from "./ReportSubTabBar";
import ReviewSummaryPanel from "./ReviewSummaryPanel";

type DocKind = "body" | "review" | "summary" | "ai_check";

const TABS: { kind: DocKind; label: string }[] = [
  { kind: "body", label: "正文" },
  { kind: "review", label: "审查" },
  { kind: "summary", label: "摘要" },
  { kind: "ai_check", label: "AI味" },
];

type Props = {
  chapter: number;
  initialTab?: DocKind;
  autoStartReview?: boolean;
  onSaved?: () => void;
  onReviewComplete?: () => void;
};

export default function ChapterDocumentPanel({
  chapter,
  initialTab = "body",
  autoStartReview = false,
  onSaved,
  onReviewComplete,
}: Props) {
  const [tab, setTab] = useState<DocKind>(initialTab);
  const [docs, setDocs] = useState<Record<DocKind, ChapterDocDTO | null>>({
    body: null,
    review: null,
    summary: null,
    ai_check: null,
  });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [hasKey, setHasKey] = useState(true);
  const [jobId, setJobId] = useState("");
  const [jobStatus, setJobStatus] = useState("");
  const [jobMessage, setJobMessage] = useState("");
  const [reviewReport, setReviewReport] = useState<ReviewReportDTO | null>(null);
  const [reviewMetrics, setReviewMetrics] = useState<ReviewMetrics | null>(null);
  const [reviewRunning, setReviewRunning] = useState(false);
  const [aiDetectMetrics, setAiDetectMetrics] = useState<AIDetectMetrics | null>(null);
  const [aiDetectRunning, setAiDetectRunning] = useState(false);
  const [aiDetectMessage, setAiDetectMessage] = useState("");
  const [aiDetectReport, setAiDetectReport] = useState<AIDetectReportDTO | null>(null);
  const [reviewSubview, setReviewSubview] = useState<ReportSubview>("summary");
  const [aiCheckSubview, setAiCheckSubview] = useState<ReportSubview>("summary");
  const jobIdRef = useRef("");
  const aiDetectJobIdRef = useRef("");
  const autoReviewStartedRef = useRef(false);

  const loadAll = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [body, review, summary, ai_check] = await Promise.all([
        app().GetChapterDocument(chapter, "body"),
        app().GetChapterDocument(chapter, "review"),
        app().GetChapterDocument(chapter, "summary"),
        app().GetChapterDocument(chapter, "ai_check"),
      ]);
      setDocs({ body, review, summary, ai_check });
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [chapter]);

  const loadReviewMetrics = useCallback(async (reviewBody: string) => {
    try {
      const dto = await app().GetChapterReviewMetrics(chapter);
      const fromText = parseReviewMetricsFromText(reviewBody);
      setReviewMetrics(mergeReviewMetrics(dto, fromText));
    } catch {
      setReviewMetrics(parseReviewMetricsFromText(reviewBody));
    }
  }, [chapter]);

  const loadAIDetectMetrics = useCallback(async (reportBody?: string) => {
    try {
      const dto = await app().GetChapterAIDetectMetrics(chapter);
      const body = reportBody ?? dto.report_body ?? docs.ai_check?.body ?? "";
      const fromText = body ? parseAIDetectMetricsFromText(body) : null;
      setAiDetectMetrics(mergeAIDetectMetrics(dto, fromText));
    } catch {
      setAiDetectMetrics(null);
    }
  }, [chapter, docs.ai_check?.body]);

  useEffect(() => {
    setTab(initialTab);
    setReviewSubview("summary");
    setAiCheckSubview("summary");
    autoReviewStartedRef.current = false;
    loadAll();
  }, [chapter, initialTab, loadAll]);

  useEffect(() => {
    if (!docs.ai_check?.exists) {
      setAiDetectMetrics(null);
      return;
    }
    void loadAIDetectMetrics(docs.ai_check.body ?? "");
  }, [docs.ai_check?.body, docs.ai_check?.exists, loadAIDetectMetrics]);

  useEffect(() => {
    if (!docs.review?.exists) {
      setReviewMetrics(null);
      return;
    }
    void loadReviewMetrics(docs.review.body ?? "");
  }, [docs.review?.body, docs.review?.exists, loadReviewMetrics]);

  useEffect(() => {
    app()
      .HasAPIKey()
      .then(setHasKey)
      .catch(() => setHasKey(false));
  }, []);

  useEffect(() => {
    void (async () => {
      try {
        const { active, job } = await app().GetActiveReviewJob();
        if (!active || !job.id || job.chapter !== chapter) return;
        jobIdRef.current = job.id;
        setJobId(job.id);
        setJobStatus(job.status);
        setReviewRunning(job.status === "pending" || job.status === "running");
      } catch {
        /* ignore */
      }
    })();
  }, [chapter]);

  useEffect(() => {
    void (async () => {
      try {
        const { active, job } = await app().GetActiveAIDetectJob();
        if (!active || !job.id || job.chapter !== chapter) return;
        aiDetectJobIdRef.current = job.id;
        setAiDetectRunning(job.status === "pending" || job.status === "running");
      } catch {
        /* ignore */
      }
    })();
  }, [chapter]);

  useEffect(() => {
    const match = (id?: string) => !jobIdRef.current || id === jobIdRef.current;
    const matchAI = (id?: string) => !aiDetectJobIdRef.current || id === aiDetectJobIdRef.current;
    const unsubs = [
      eventsOn(REVIEW_EVENTS.status, (p) => {
        if (p.chapter !== chapter || !match(p.job_id)) return;
        setJobStatus(p.status || "");
        setJobMessage(p.message || "");
        setReviewRunning(p.status === "pending" || p.status === "running");
      }),
      eventsOn(REVIEW_EVENTS.done, (p) => {
        if (p.chapter !== chapter || !match(p.job_id)) return;
        setReviewRunning(false);
        setJobStatus("done");
        if (p.report) {
          try {
            setReviewReport(JSON.parse(p.report) as ReviewReportDTO);
          } catch {
            setReviewReport(null);
          }
        }
        void loadAll().then(() => {
          setReviewSubview("summary");
          setTab("review");
          onReviewComplete?.();
        });
      }),
      eventsOn(REVIEW_EVENTS.error, (p) => {
        if (p.chapter !== chapter || !match(p.job_id)) return;
        setReviewRunning(false);
        setJobStatus("failed");
        setError(p.error || "审查失败");
      }),
      eventsOn(AI_DETECT_EVENTS.status, (p) => {
        if (p.chapter !== chapter || !matchAI(p.job_id)) return;
        setAiDetectRunning(p.status === "pending" || p.status === "running");
        setAiDetectMessage(p.message || "");
      }),
      eventsOn(AI_DETECT_EVENTS.done, (p) => {
        if (p.chapter !== chapter || !matchAI(p.job_id)) return;
        setAiDetectRunning(false);
        if (p.report) {
          try {
            setAiDetectReport(JSON.parse(p.report) as AIDetectReportDTO);
          } catch {
            setAiDetectReport(null);
          }
        }
        void loadAll().then(() => {
          setAiCheckSubview("summary");
          setTab("ai_check");
        });
      }),
      eventsOn(AI_DETECT_EVENTS.error, (p) => {
        if (p.chapter !== chapter || !matchAI(p.job_id)) return;
        setAiDetectRunning(false);
        setError(p.error || "AI 味检测失败");
      }),
    ];
    return () => unsubs.forEach((u) => u());
  }, [chapter, loadAll, onReviewComplete]);

  const startAIDetect = useCallback(async () => {
    if (!docs.body?.exists) {
      setError("请先完成本章正文再检测");
      return;
    }
    setError("");
    setAiDetectReport(null);
    setAiDetectMessage("");
    try {
      const job = await app().StartAIDetectChapter({ chapter });
      aiDetectJobIdRef.current = job.id;
      setAiDetectRunning(true);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, [chapter, docs.body?.exists]);

  const cancelAIDetect = async () => {
    if (!aiDetectJobIdRef.current) return;
    try {
      await app().CancelAIDetectChapter(aiDetectJobIdRef.current);
      setAiDetectRunning(false);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const startReview = useCallback(async () => {
    if (!docs.body?.exists) {
      setError("请先完成本章正文再审查");
      return;
    }
    setError("");
    setReviewReport(null);
    setJobMessage("");
    try {
      const job = await app().StartReviewChapter({ chapter });
      jobIdRef.current = job.id;
      setJobId(job.id);
      setJobStatus(job.status);
      setReviewRunning(true);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, [chapter, docs.body?.exists]);

  useEffect(() => {
    if (!autoStartReview || autoReviewStartedRef.current || loading || reviewRunning) return;
    autoReviewStartedRef.current = true;
    void startReview();
  }, [autoStartReview, loading, reviewRunning, startReview]);

  const cancelReview = async () => {
    if (!jobId) return;
    try {
      await app().CancelReviewChapter(jobId);
      setReviewRunning(false);
      setJobStatus("cancelled");
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const switchReportSubview = async (
    next: ReportSubview,
    currentSubview: ReportSubview,
    setSubview: (v: ReportSubview) => void,
  ) => {
    if (next === currentSubview) return;
    if (currentSubview === "full") {
      const ok = await confirmUnsavedLeave();
      if (!ok) return;
    }
    setSubview(next);
  };

  const goToAIDetectTab = async (subview: ReportSubview = "summary") => {
    if (tab === "ai_check") return;
    const ok = await confirmUnsavedLeave();
    if (!ok) return;
    setAiCheckSubview(subview);
    setTab("ai_check");
  };

  const current = docs[tab];
  const editorValue =
    tab === "body" ? normalizeChapterBodyForDisplay(current?.body ?? "") : (current?.body ?? "");

  const save = async (body: string) => {
    setSaving(true);
    try {
      await app().SaveChapterDocument(chapter, tab, body);
      setDocs((prev) => ({
        ...prev,
        [tab]: prev[tab]
          ? { ...prev[tab]!, body, exists: true }
          : { kind: tab, chapter, title: "", body, exists: true },
      }));
      if (tab === "review") {
        void loadReviewMetrics(body);
      }
      if (tab === "ai_check") {
        void loadAIDetectMetrics(body);
      }
      onSaved?.();
    } finally {
      setSaving(false);
    }
  };

  const switchDocTab = async (kind: DocKind) => {
    if (kind === tab) return;
    const ok = await confirmUnsavedLeave();
    if (!ok) return;
    if (kind === "review") setReviewSubview("summary");
    if (kind === "ai_check") setAiCheckSubview("summary");
    setTab(kind);
  };

  if (loading) {
    return (
      <div className="flex flex-1 items-center justify-center text-studio-muted">
        <Loader2 className="h-6 w-6 animate-spin" />
      </div>
    );
  }

  const bodyReady = docs.body?.exists;
  const tabExists = (kind: DocKind) => docs[kind]?.exists;

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
      <div className="flex shrink-0 flex-wrap items-center justify-between gap-2 border-b border-studio-border px-3 py-2">
        <div className="flex gap-1">
          {TABS.map(({ kind, label }) => {
            const exists = tabExists(kind);
            return (
              <button
                key={kind}
                type="button"
                onClick={() => switchDocTab(kind)}
                className={`rounded-lg px-3 py-1.5 text-xs transition ${
                  tab === kind
                    ? kind === "ai_check"
                      ? "bg-studio-ai/15 text-studio-ai"
                      : "bg-studio-accent/15 text-studio-accent"
                    : "text-studio-muted hover:bg-studio-bg hover:text-studio-text"
                }`}
              >
                {label}
                {!exists && kind !== "body" && (
                  <span className="ml-1 text-studio-muted/50">·</span>
                )}
              </button>
            );
          })}
        </div>

        <div className="flex items-center gap-2">
          {aiDetectMetrics && !aiDetectRunning && (
            <button
              type="button"
              onClick={() => void goToAIDetectTab()}
              className={`hidden rounded-full bg-studio-panel px-2 py-0.5 text-[10px] font-medium sm:inline hover:ring-1 hover:ring-studio-ai/30 ${
                aiDetectMetrics.aiScore != null ? aiScoreColor(aiDetectMetrics.aiScore) : "text-studio-muted"
              }`}
              title="查看完整 AI 味报告"
            >
              AI {aiDetectMetrics.aiScore?.toFixed(1) ?? "—"} · {riskLevelLabel(aiDetectMetrics.riskLevel)}
            </button>
          )}
          {aiDetectRunning ? (
            <>
              <span className="inline-flex items-center gap-1.5 text-xs text-studio-muted">
                <Loader2 className="h-3.5 w-3.5 animate-spin text-studio-ai" />
                {aiDetectMessage || "检测 AI 味…"}
              </span>
              <button
                type="button"
                onClick={() => void cancelAIDetect()}
                className="inline-flex items-center gap-1 rounded-lg border border-studio-border px-2.5 py-1 text-xs hover:bg-studio-bg"
              >
                <Square className="h-3 w-3" />
                取消
              </button>
            </>
          ) : (
            <button
              type="button"
              onClick={() => void startAIDetect()}
              disabled={!hasKey || !bodyReady}
              title={!bodyReady ? "需要先有正文" : "检测本章是否有明显 AI 生成痕迹"}
              className="inline-flex items-center gap-1.5 rounded-lg border border-studio-border bg-studio-panel px-3 py-1.5 text-xs font-medium text-studio-text hover:bg-studio-bg disabled:opacity-50"
            >
              <Bot className="h-3.5 w-3.5 text-studio-ai" />
              AI味判断
            </button>
          )}
          {reviewRunning ? (
            <>
              <span className="inline-flex items-center gap-1.5 text-xs text-studio-muted">
                <Loader2 className="h-3.5 w-3.5 animate-spin text-studio-accent" />
                {jobMessage || "正在审查…"}
              </span>
              <button
                type="button"
                onClick={() => void cancelReview()}
                className="inline-flex items-center gap-1 rounded-lg border border-studio-border px-2.5 py-1 text-xs hover:bg-studio-bg"
              >
                <Square className="h-3 w-3" />
                取消
              </button>
            </>
          ) : (
            <button
              type="button"
              onClick={() => void startReview()}
              disabled={!hasKey || !bodyReady}
              title={!bodyReady ? "需要先有正文" : undefined}
              className="inline-flex items-center gap-1.5 rounded-lg bg-studio-accent px-3 py-1.5 text-xs font-medium text-studio-on-accent hover:brightness-110 disabled:opacity-50"
            >
              <ClipboardCheck className="h-3.5 w-3.5" />
              {docs.review?.exists ? "重新审查" : "AI 审查"}
            </button>
          )}
        </div>
      </div>

      {!hasKey && (
        <div className="mx-3 mt-2 shrink-0 studio-alert-error-compact">
          请先在设置中配置 API Key 后再使用 AI 审查。
        </div>
      )}

      {error && <div className="mx-3 mt-2 shrink-0 studio-alert-error-compact">{error}</div>}

      {aiDetectReport && !aiDetectRunning && tab !== "ai_check" && (
        <button
          type="button"
          onClick={() => void goToAIDetectTab()}
          className="mx-3 mt-2 shrink-0 rounded-lg border border-studio-ai/30 bg-studio-ai/5 px-3 py-2 text-left text-sm text-studio-text transition hover:bg-studio-ai/10"
        >
          <span className="flex items-center justify-between gap-2">
            <span>{aiDetectReport.summary}</span>
            <span className="inline-flex shrink-0 items-center gap-0.5 text-xs text-studio-ai">
              查看完整报告
              <ChevronRight className="h-3.5 w-3.5" />
            </span>
          </span>
        </button>
      )}

      {reviewReport && !reviewRunning && jobStatus === "done" && tab !== "review" && (
        <button
          type="button"
          onClick={() => {
            void (async () => {
              const ok = await confirmUnsavedLeave();
              if (!ok) return;
              setReviewSubview("summary");
              setTab("review");
            })();
          }}
          className="mx-3 mt-2 shrink-0 rounded-lg border border-[rgb(var(--studio-diff-add-border))] bg-[rgb(var(--studio-diff-add-bg))] px-3 py-2 text-left text-sm text-[rgb(var(--studio-diff-add-stat))] transition hover:brightness-[0.98]"
        >
          <span className="flex items-center justify-between gap-2">
            <span>{reviewReport.summary}</span>
            <span className="inline-flex shrink-0 items-center gap-0.5 text-xs">
              查看审查摘要
              <ChevronRight className="h-3.5 w-3.5" />
            </span>
          </span>
        </button>
      )}

      {tab !== "body" && current && !current.exists && !reviewRunning && !aiDetectRunning && (
        <p className="shrink-0 px-4 py-2 text-xs text-studio-muted">
          {tab === "review" && "暂无审查报告，可点击「AI 审查」生成，或在编辑模式下手动新建并保存。"}
          {tab === "ai_check" && "暂无 AI 味报告，可点击「AI味判断」生成。"}
          {tab === "summary" && "暂无摘要，可在编辑模式下手动新建并保存。"}
        </p>
      )}

      <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
        {tab === "review" && docs.review?.exists && (
          <>
            <ReportSubTabBar
              value={reviewSubview}
              onChange={(next) => void switchReportSubview(next, reviewSubview, setReviewSubview)}
            />
            {reviewSubview === "summary" ? (
              reviewMetrics ? (
                <ReviewSummaryPanel metrics={reviewMetrics} fill />
              ) : (
                <div className="flex flex-1 items-center justify-center px-4 text-xs text-studio-muted">
                  暂无结构化摘要，请查看「完整报告」
                </div>
              )
            ) : (
              <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
                <MarkdownEditor
                  key={`${chapter}-review-full`}
                  value={docs.review.body ?? ""}
                  paper
                  saving={saving}
                  onSave={save}
                  relaxed
                  emptyHint="暂无审查报告"
                />
              </div>
            )}
          </>
        )}

        {tab === "ai_check" && docs.ai_check?.exists && (
          <>
            <ReportSubTabBar
              value={aiCheckSubview}
              onChange={(next) => void switchReportSubview(next, aiCheckSubview, setAiCheckSubview)}
              accent="ai"
            />
            {aiCheckSubview === "summary" ? (
              aiDetectMetrics ? (
                <AIDetectSummaryPanel metrics={aiDetectMetrics} fill />
              ) : (
                <div className="flex flex-1 items-center justify-center px-4 text-xs text-studio-muted">
                  暂无结构化摘要，请查看「完整报告」
                </div>
              )
            ) : (
              <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
                <MarkdownEditor
                  key={`${chapter}-ai_check-full`}
                  value={docs.ai_check.body ?? ""}
                  paper
                  saving={saving}
                  onSave={save}
                  relaxed
                  emptyHint="暂无 AI 味报告"
                />
              </div>
            )}
          </>
        )}

        {(tab === "body" || tab === "summary" || (tab === "review" && !docs.review?.exists) || (tab === "ai_check" && !docs.ai_check?.exists)) && (
          <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
            <MarkdownEditor
              key={`${chapter}-${tab}`}
              value={editorValue}
              paper
              saving={saving}
              onSave={save}
              selectionChapter={tab === "body" ? chapter : undefined}
              relaxed={tab === "summary"}
              emptyHint={
                tab === "body"
                  ? "正文为空"
                  : tab === "summary"
                    ? "暂无摘要"
                    : tab === "review"
                      ? "暂无审查报告"
                      : "暂无 AI 味报告"
              }
            />
          </div>
        )}
      </div>
    </div>
  );
}
