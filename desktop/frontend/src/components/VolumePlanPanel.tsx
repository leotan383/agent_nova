import { useCallback, useEffect, useRef, useState } from "react";
import { Check, Loader2, RefreshCw, Sparkles, Square, X } from "lucide-react";
import { PLAN_EVENTS, eventsOn } from "../lib/runtime";
import {
  DiffResultDTO,
  PlanReportDTO,
  ReplanResultDTO,
  VolumeOutlineDTO,
  ChapterDTO,
  app,
} from "../lib/wails";
import { confirmUnsavedLeave } from "../lib/unsavedGuard";
import ChapterDiffView from "./ChapterDiffView";
import ChapterStructureDialog from "./ChapterStructureDialog";
import MarkdownEditor from "./MarkdownEditor";
import OutlineChapterMatrix from "./OutlineChapterMatrix";
import BookReadReportPanel from "./BookReadReportPanel";
import BatchPolishDialog from "./BatchPolishDialog";
import PolishChapterPicker from "./PolishChapterPicker";

type Props = {
  suggestedVolume: number;
  currentChapter?: number;
  focusVolume?: number | null;
  focusPlanView?: PlanView | null;
  onFocusApplied?: () => void;
  chapters?: ChapterDTO[];
  structureRefreshKey?: number;
  onComplete: () => void;
  onOpenChapter?: (chapter: number) => void;
  onStructureChange?: () => void;
};

type PlanView = "edit" | "matrix" | "tools";

export default function VolumePlanPanel({
  suggestedVolume,
  currentChapter = 0,
  focusVolume,
  focusPlanView,
  onFocusApplied,
  chapters = [],
  structureRefreshKey = 0,
  onComplete,
  onOpenChapter,
  onStructureChange,
}: Props) {
  const [planView, setPlanView] = useState<PlanView>("edit");
  const [polishChapters, setPolishChapters] = useState<number[]>([]);
  const [structureOpen, setStructureOpen] = useState(false);
  const [structureChapter, setStructureChapter] = useState(0);
  const [structureHint, setStructureHint] = useState("");
  const [volume, setVolume] = useState(Math.max(1, suggestedVolume));
  const [volumeInput, setVolumeInput] = useState(String(Math.max(1, suggestedVolume)));
  const [outline, setOutline] = useState<VolumeOutlineDTO | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [hasKey, setHasKey] = useState(true);
  const [jobId, setJobId] = useState("");
  const [jobStatus, setJobStatus] = useState("");
  const [jobMessage, setJobMessage] = useState("");
  const [jobKind, setJobKind] = useState("");
  const [report, setReport] = useState<PlanReportDTO | null>(null);
  const [running, setRunning] = useState(false);
  const [replanNotes, setReplanNotes] = useState("");
  const [replanResult, setReplanResult] = useState<ReplanResultDTO | null>(null);
  const [previewDiff, setPreviewDiff] = useState<DiffResultDTO | null>(null);
  const [showDiffModal, setShowDiffModal] = useState(false);
  const [applyingReplan, setApplyingReplan] = useState(false);
  const jobIdRef = useRef("");
  const panelRef = useRef<HTMLDivElement>(null);

  const canReplan = (outline?.exists ?? false) && currentChapter > 0;

  useEffect(() => {
    const v = Math.max(1, suggestedVolume);
    setVolume(v);
    setVolumeInput(String(v));
  }, [suggestedVolume]);

  useEffect(() => {
    if (!focusVolume || focusVolume <= 0) return;
    void (async () => {
      if (focusVolume !== volume) {
        const ok = await confirmUnsavedLeave();
        if (!ok) return;
        setVolume(focusVolume);
        setVolumeInput(String(focusVolume));
      }
      if (focusPlanView) {
        setPlanView(focusPlanView);
      }
      onFocusApplied?.();
      panelRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
    })();
  }, [focusVolume, focusPlanView, volume, onFocusApplied]);

  useEffect(() => {
    if (!focusPlanView || (focusVolume && focusVolume > 0)) return;
    setPlanView(focusPlanView);
    onFocusApplied?.();
  }, [focusPlanView, focusVolume, onFocusApplied]);

  useEffect(() => {
    app()
      .HasAPIKey()
      .then(setHasKey)
      .catch(() => setHasKey(false));
  }, []);

  const loadOutline = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const o = await app().GetVolumeOutline(volume);
      setOutline(o);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setOutline(null);
    } finally {
      setLoading(false);
    }
  }, [volume]);

  useEffect(() => {
    loadOutline();
  }, [loadOutline]);

  useEffect(() => {
    void (async () => {
      try {
        const { active, job } = await app().GetActivePlanJob();
        if (!active || !job.id) return;
        jobIdRef.current = job.id;
        setJobId(job.id);
        setJobStatus(job.status);
        setJobKind(job.kind ?? "plan");
        setRunning(job.status === "pending" || job.status === "running");
        if (job.volume > 0 && job.volume !== volume) {
          setVolume(job.volume);
          setVolumeInput(String(job.volume));
        }
      } catch {
        /* ignore */
      }
    })();
  }, []);

  const openReplanPreview = useCallback(async (result: ReplanResultDTO) => {
    setReplanResult(result);
    try {
      const diff = await app().PreviewVolumeOutlineDiff(volume, result.proposed_body);
      setPreviewDiff(diff);
      setShowDiffModal(true);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, [volume]);

  useEffect(() => {
    const match = (id?: string) => !jobIdRef.current || id === jobIdRef.current;
    const unsubs = [
      eventsOn(PLAN_EVENTS.status, (p) => {
        if (!match(p.job_id)) return;
        setJobStatus(p.status || "");
        setJobMessage(p.message || "");
        setRunning(p.status === "pending" || p.status === "running");
      }),
      eventsOn(PLAN_EVENTS.done, (p) => {
        if (!match(p.job_id)) return;
        setRunning(false);
        setJobStatus("done");
        const kind = (p as { kind?: string }).kind ?? "plan";
        if (kind === "replan" && (p as { replan?: string }).replan) {
          try {
            const result = JSON.parse((p as { replan: string }).replan) as ReplanResultDTO;
            setReport(null);
            void openReplanPreview(result);
          } catch {
            setError("Replan 结果解析失败");
          }
          return;
        }
        if (p.report) {
          try {
            setReport(JSON.parse(p.report) as PlanReportDTO);
          } catch {
            setReport(null);
          }
        }
        loadOutline();
        onComplete();
      }),
      eventsOn(PLAN_EVENTS.error, (p) => {
        if (!match(p.job_id)) return;
        setRunning(false);
        setJobStatus("failed");
        setError(p.error || "卷纲任务失败");
      }),
    ];
    return () => unsubs.forEach((u) => u());
  }, [loadOutline, onComplete, openReplanPreview]);

  const startPlan = async () => {
    setError("");
    setReport(null);
    setReplanResult(null);
    setJobMessage("");
    setJobKind("plan");
    try {
      const job = await app().StartPlanVolume({ volume });
      jobIdRef.current = job.id;
      setJobId(job.id);
      setJobStatus(job.status);
      setRunning(true);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const startReplan = async () => {
    setError("");
    setReport(null);
    setReplanResult(null);
    setJobMessage("");
    setJobKind("replan");
    try {
      const job = await app().StartReplanVolume({
        volume,
        from_chapter: currentChapter + 1,
        notes: replanNotes.trim(),
      });
      jobIdRef.current = job.id;
      setJobId(job.id);
      setJobStatus(job.status);
      setRunning(true);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const cancelPlan = async () => {
    if (!jobId) return;
    try {
      await app().CancelPlanVolume(jobId);
      setRunning(false);
      setJobStatus("cancelled");
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const applyReplan = async () => {
    if (!replanResult?.proposed_body) return;
    setApplyingReplan(true);
    setError("");
    try {
      await app().SaveVolumeOutline(volume, replanResult.proposed_body);
      setShowDiffModal(false);
      setPreviewDiff(null);
      setReplanResult(null);
      await loadOutline();
      onComplete();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setApplyingReplan(false);
    }
  };

  const saveOutline = async (body: string) => {
    setSaving(true);
    setError("");
    try {
      await app().SaveVolumeOutline(volume, body);
      setOutline((prev) => ({
        volume,
        path: prev?.path ?? "",
        body,
        exists: true,
      }));
      onComplete();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      throw e;
    } finally {
      setSaving(false);
    }
  };

  const commitVolume = async () => {
    const next = Math.max(1, parseInt(volumeInput, 10) || 1);
    setVolumeInput(String(next));
    if (next === volume) return;
    const ok = await confirmUnsavedLeave();
    if (!ok) {
      setVolumeInput(String(volume));
      return;
    }
    setVolume(next);
  };

  return (
    <div
      ref={panelRef}
      className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl border border-studio-border bg-studio-panel p-5"
    >
      <div className="mb-4 flex shrink-0 flex-col gap-2">
        <div>
          <h3 className="text-sm font-medium">卷纲规划</h3>
          <p className="mt-1 text-xs text-studio-muted">
            基于总纲与设定生成详细卷纲
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2 rounded-lg border border-studio-border bg-studio-bg/40 px-3 py-2">
          <label className="flex shrink-0 items-center gap-2 text-sm">
            <span className="text-studio-muted">卷号</span>
            <input
              type="number"
              min={1}
              value={volumeInput}
              disabled={running}
              onChange={(e) => setVolumeInput(e.target.value)}
              onBlur={() => void commitVolume()}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  void commitVolume();
                }
              }}
              className="w-16 rounded-lg border border-studio-border bg-studio-bg px-2 py-1 text-center"
            />
          </label>

          <div className="ml-auto flex flex-wrap items-center gap-2">
            {running ? (
              <button
                type="button"
                onClick={cancelPlan}
                className="inline-flex items-center gap-1.5 rounded-lg border border-studio-border px-3 py-1.5 text-sm hover:bg-studio-bg"
              >
                <Square className="h-3.5 w-3.5" />
                取消
              </button>
            ) : (
              <>
                {canReplan && (
                  <button
                    type="button"
                    onClick={() => void startReplan()}
                    disabled={!hasKey || loading}
                    className="inline-flex items-center gap-1.5 rounded-lg border border-studio-accent/40 px-3 py-1.5 text-sm text-studio-accent hover:bg-studio-accent/10 disabled:opacity-50"
                    title={`从第 ${currentChapter + 1} 章起重新规划`}
                  >
                    <RefreshCw className="h-3.5 w-3.5" />
                    Replan
                  </button>
                )}
                <button
                  type="button"
                  onClick={() => void startPlan()}
                  disabled={!hasKey || loading}
                  className="inline-flex items-center gap-1.5 rounded-lg bg-studio-accent px-3 py-1.5 text-sm font-medium text-white hover:opacity-90 disabled:opacity-50"
                  title={outline?.exists ? "整卷重新生成卷纲（直接覆盖）" : "基于总纲与设定生成卷纲"}
                >
                  {loading ? (
                    <Loader2 className="h-3.5 w-3.5 animate-spin" />
                  ) : (
                    <Sparkles className="h-3.5 w-3.5" />
                  )}
                  {outline?.exists ? "重新生成" : "AI 生成卷纲"}
                </button>
              </>
            )}
          </div>
        </div>

        {canReplan && !running && (
          <p className="rounded-md bg-studio-accent/5 px-3 py-1.5 text-[11px] leading-relaxed text-studio-muted">
            <span className="font-medium text-studio-text/80">Replan</span>
            {" "}保留已写 {currentChapter} 章，从第 {currentChapter + 1} 章起调整后续章纲，预览 diff 后应用
          </p>
        )}
      </div>

      <div className="mb-3 flex shrink-0 gap-1 rounded-lg border border-studio-border bg-studio-bg/50 p-1">
        {(
          [
            ["edit", "卷纲编辑"],
            ["matrix", "对照视图"],
            ["tools", "全书工具"],
          ] as const
        ).map(([id, label]) => (
          <button
            key={id}
            type="button"
            onClick={() => setPlanView(id)}
            className={`flex-1 rounded-md px-3 py-1.5 text-xs font-medium transition ${
              planView === id
                ? "bg-studio-panel text-studio-accent shadow-sm"
                : "text-studio-muted hover:text-studio-text"
            }`}
          >
            {label}
          </button>
        ))}
      </div>

      {planView === "matrix" && (
        <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
          {structureHint && (
            <div className="mb-2 shrink-0 rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-800 dark:text-amber-200">
              {structureHint}
            </div>
          )}
          <OutlineChapterMatrix
            volume={volume}
            refreshKey={structureRefreshKey}
            onOpenChapter={onOpenChapter}
            onStartReplan={() => void startReplan()}
            onInsertAfter={(after) => {
              setStructureChapter(after);
              setStructureOpen(true);
            }}
          />
        </div>
      )}

      {planView === "tools" && (
        <div className="min-h-0 flex-1 space-y-4 overflow-y-auto">
          <div>
            <h4 className="mb-2 text-xs font-medium text-studio-muted">全书通读报告</h4>
            <BookReadReportPanel currentChapter={currentChapter} onOpenChapter={onOpenChapter} />
          </div>
          <div>
            <h4 className="mb-2 text-xs font-medium text-studio-muted">批量润色</h4>
            <p className="mb-2 text-[11px] leading-relaxed text-studio-muted">
              统一人称、称谓或语气，不改变情节。先勾选章节，预览 diff 后再逐章应用。
            </p>
            <PolishChapterPicker
              chapters={chapters}
              refreshKey={structureRefreshKey}
              selected={polishChapters}
              onChange={setPolishChapters}
            />
            <div className="mt-3">
              <BatchPolishDialog chapters={polishChapters} onApplied={onStructureChange} />
            </div>
          </div>
        </div>
      )}

      {planView === "edit" && (
        <>
      {canReplan && !running && (
        <div className="mb-3">
          <label className="block text-xs text-studio-muted">
            Replan 备注（可选）
            <input
              type="text"
              value={replanNotes}
              onChange={(e) => setReplanNotes(e.target.value)}
              placeholder="例如：加强反派线、压缩中段节奏…"
              className="mt-1 w-full rounded-lg border border-studio-border bg-studio-bg px-3 py-1.5 text-sm outline-none"
            />
          </label>
        </div>
      )}

      {!hasKey && (
        <div className="mb-3 studio-alert-error-compact">
          请先在设置中配置 API Key 后再生成卷纲。
        </div>
      )}

      {error && <div className="mb-3 studio-alert-error-compact">{error}</div>}

      {running && (
        <div className="mb-3 flex flex-col gap-1 rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm">
          <div className="flex items-center gap-2">
            <Loader2 className="h-4 w-4 animate-spin text-studio-accent" />
            <span>
              {jobMessage ||
                (jobKind === "replan" ? "正在 Replan 卷纲…" : "正在生成卷纲…")}
            </span>
            {jobStatus && (
              <span className="text-xs text-studio-muted">({jobStatus})</span>
            )}
          </div>
          <p className="text-[11px] text-studio-muted/80">
            卷纲任务进行中。切换 Tab 后返回此页可继续查看进度。
          </p>
        </div>
      )}

      {report && !running && (
        <div className="mb-3 rounded-lg border border-[rgb(var(--studio-diff-add-border))] bg-[rgb(var(--studio-diff-add-bg))] px-3 py-2 text-sm text-[rgb(var(--studio-diff-add-stat))]">
          {report.summary}
        </div>
      )}

      {replanResult && !showDiffModal && !running && (
        <div className="mb-3 rounded-lg border border-studio-accent/30 bg-studio-accent/10 px-3 py-2 text-sm">
          <p>{replanResult.summary}</p>
          <button
            type="button"
            onClick={() => void openReplanPreview(replanResult)}
            className="mt-2 text-xs font-medium text-studio-accent hover:underline"
          >
            查看 diff 并确认应用
          </button>
        </div>
      )}

      {loading ? (
        <div className="flex min-h-[240px] items-center justify-center text-sm text-studio-muted">
          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          加载卷纲…
        </div>
      ) : (
        <div className="flex min-h-0 max-h-[min(560px,60vh)] flex-1 flex-col overflow-hidden rounded-lg border border-studio-border">
          <MarkdownEditor
            value={outline?.body ?? ""}
            editable
            saving={saving}
            onSave={saveOutline}
            emptyHint={
              outline?.exists
                ? "卷纲为空"
                : "尚无卷纲，点击「AI 生成卷纲」或手动编辑后保存"
            }
          />
        </div>
      )}
        </>
      )}

      {showDiffModal && previewDiff && (
        <div className="studio-modal-overlay fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="flex max-h-[85vh] w-full max-w-2xl flex-col overflow-hidden rounded-xl border border-studio-border bg-studio-panel shadow-card">
            <div className="flex shrink-0 items-center justify-between border-b border-studio-border px-4 py-3">
              <div>
                <span className="text-sm font-medium">Replan 卷纲 · 变更预览</span>
                {replanResult && (
                  <p className="mt-0.5 text-xs text-studio-muted">
                    已写至第 {replanResult.written_through} 章 · 从第 {replanResult.from_chapter} 章起调整
                  </p>
                )}
              </div>
              <button
                type="button"
                onClick={() => setShowDiffModal(false)}
                className="rounded p-1 text-studio-muted hover:bg-studio-bg hover:text-studio-text"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
            <div className="min-h-0 flex-1 overflow-y-auto p-4">
              <ChapterDiffView diff={previewDiff} maxHeight="max-h-[55vh]" />
            </div>
            <div className="flex shrink-0 gap-2 border-t border-studio-border p-3">
              <button
                type="button"
                onClick={() => setShowDiffModal(false)}
                className="flex-1 rounded-lg border border-studio-border px-3 py-2 text-sm text-studio-muted hover:bg-studio-bg"
              >
                取消
              </button>
              <button
                type="button"
                onClick={() => void applyReplan()}
                disabled={applyingReplan}
                className="inline-flex flex-1 items-center justify-center gap-1 rounded-lg bg-studio-accent px-3 py-2 text-sm font-medium text-studio-on-accent hover:brightness-110 disabled:opacity-40"
              >
                {applyingReplan ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Check className="h-4 w-4" />
                )}
                确认应用新卷纲
              </button>
            </div>
          </div>
        </div>
      )}

      <ChapterStructureDialog
        open={structureOpen}
        mode="insert"
        chapter={structureChapter}
        onClose={() => setStructureOpen(false)}
        onDone={() => {
          setStructureHint("章节结构已更新。卷纲 Markdown 不会自动改章号，建议在对照视图发起 Replan 或手动编辑卷纲。");
          onStructureChange?.();
          onComplete();
        }}
      />
    </div>
  );
}
