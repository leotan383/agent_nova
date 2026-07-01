import { useCallback, useEffect, useRef, useState } from "react";
import { ClipboardCheck, Loader2, Square } from "lucide-react";
import { REVIEW_EVENTS, eventsOn } from "../lib/runtime";
import { stripReviewMetricsSuffix } from "../lib/chapterBody";
import { mergeReviewMetrics, parseReviewMetricsFromText, ReviewMetrics } from "../lib/reviewMetrics";
import { ChapterDocDTO, ReviewReportDTO, app } from "../lib/wails";
import { confirmUnsavedLeave } from "../lib/unsavedGuard";
import MarkdownEditor from "./MarkdownEditor";
import ReviewSummaryPanel from "./ReviewSummaryPanel";

type DocKind = "body" | "review" | "summary";

const TABS: { kind: DocKind; label: string }[] = [
  { kind: "body", label: "正文" },
  { kind: "review", label: "审查" },
  { kind: "summary", label: "摘要" },
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
  const jobIdRef = useRef("");
  const autoReviewStartedRef = useRef(false);

  const loadAll = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [body, review, summary] = await Promise.all([
        app().GetChapterDocument(chapter, "body"),
        app().GetChapterDocument(chapter, "review"),
        app().GetChapterDocument(chapter, "summary"),
      ]);
      setDocs({ body, review, summary });
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

  useEffect(() => {
    setTab(initialTab);
    autoReviewStartedRef.current = false;
    loadAll();
  }, [chapter, initialTab, loadAll]);

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
    const match = (id?: string) => !jobIdRef.current || id === jobIdRef.current;
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
    ];
    return () => unsubs.forEach((u) => u());
  }, [chapter, loadAll, onReviewComplete]);

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

  const current = docs[tab];
  const editorValue =
    tab === "body" ? stripReviewMetricsSuffix(current?.body ?? "") : (current?.body ?? "");

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
      onSaved?.();
    } finally {
      setSaving(false);
    }
  };

  const switchDocTab = async (kind: DocKind) => {
    if (kind === tab) return;
    const ok = await confirmUnsavedLeave();
    if (!ok) return;
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

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
      <div className="flex shrink-0 flex-wrap items-center justify-between gap-2 border-b border-studio-border px-3 py-2">
        <div className="flex gap-1">
          {TABS.map(({ kind, label }) => {
            const exists = docs[kind]?.exists;
            return (
              <button
                key={kind}
                type="button"
                onClick={() => switchDocTab(kind)}
                className={`rounded-lg px-3 py-1.5 text-xs transition ${
                  tab === kind
                    ? "bg-studio-accent/15 text-studio-accent"
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

      {reviewReport && !reviewRunning && jobStatus === "done" && (
        <div className="mx-3 mt-2 shrink-0 rounded-lg border border-[rgb(var(--studio-diff-add-border))] bg-[rgb(var(--studio-diff-add-bg))] px-3 py-2 text-sm text-[rgb(var(--studio-diff-add-stat))]">
          {reviewReport.summary}
        </div>
      )}

      {tab !== "body" && current && !current.exists && !reviewRunning && (
        <p className="shrink-0 px-4 py-2 text-xs text-studio-muted">
          暂无{tab === "review" ? "审查报告" : "摘要"}，可点击「AI 审查」生成，或在编辑模式下手动新建并保存。
        </p>
      )}

      <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
        {tab === "review" && reviewMetrics && (
          <ReviewSummaryPanel metrics={reviewMetrics} compact />
        )}
        <MarkdownEditor
          key={`${chapter}-${tab}`}
          value={editorValue}
          paper
          saving={saving}
          onSave={save}
          selectionChapter={tab === "body" ? chapter : undefined}
          emptyHint={
            tab === "body"
              ? "正文为空"
              : tab === "review"
                ? "暂无审查报告"
                : "暂无摘要"
          }
        />
      </div>
    </div>
  );
}
