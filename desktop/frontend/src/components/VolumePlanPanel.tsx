import { useCallback, useEffect, useRef, useState } from "react";
import { Loader2, Sparkles, Square } from "lucide-react";
import { PLAN_EVENTS, eventsOn } from "../lib/runtime";
import { PlanReportDTO, VolumeOutlineDTO, app } from "../lib/wails";
import { confirmUnsavedLeave } from "../lib/unsavedGuard";
import MarkdownEditor from "./MarkdownEditor";

type Props = {
  suggestedVolume: number;
  focusVolume?: number | null;
  onComplete: () => void;
};

export default function VolumePlanPanel({ suggestedVolume, focusVolume, onComplete }: Props) {
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
  const [report, setReport] = useState<PlanReportDTO | null>(null);
  const [running, setRunning] = useState(false);
  const jobIdRef = useRef("");
  const panelRef = useRef<HTMLDivElement>(null);

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
      panelRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
    })();
  }, [focusVolume, volume]);

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
        setError(p.error || "卷纲生成失败");
      }),
    ];
    return () => unsubs.forEach((u) => u());
  }, [loadOutline, onComplete]);

  const startPlan = async () => {
    setError("");
    setReport(null);
    setJobMessage("");
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
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 className="text-sm font-medium">卷纲规划</h3>
          <p className="mt-1 text-xs text-studio-muted">
            基于总纲与设定生成详细卷纲，写章前章纲将从此提取。
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <label className="flex items-center gap-2 text-sm">
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
            <button
              type="button"
              onClick={startPlan}
              disabled={!hasKey || loading}
              className="inline-flex items-center gap-1.5 rounded-lg bg-studio-accent px-3 py-1.5 text-sm font-medium text-white hover:opacity-90 disabled:opacity-50"
            >
              {loading ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
              ) : (
                <Sparkles className="h-3.5 w-3.5" />
              )}
              {outline?.exists ? "重新生成" : "AI 生成卷纲"}
            </button>
          )}
        </div>
      </div>

      {!hasKey && (
        <div className="mb-3 studio-alert-error-compact">
          请先在设置中配置 API Key 后再生成卷纲。
        </div>
      )}

      {error && <div className="mb-3 studio-alert-error-compact">{error}</div>}

      {running && (
        <div className="mb-3 flex items-center gap-2 rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm">
          <Loader2 className="h-4 w-4 animate-spin text-studio-accent" />
          <span>{jobMessage || "正在生成卷纲…"}</span>
          {jobStatus && (
            <span className="text-xs text-studio-muted">({jobStatus})</span>
          )}
        </div>
      )}

      {report && !running && (
        <div className="mb-3 rounded-lg border border-[rgb(var(--studio-diff-add-border))] bg-[rgb(var(--studio-diff-add-bg))] px-3 py-2 text-sm text-[rgb(var(--studio-diff-add-stat))]">
          {report.summary}
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
    </div>
  );
}
