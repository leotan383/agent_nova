import { useCallback, useEffect, useState } from "react";
import {
  AlertTriangle,
  Archive,
  Check,
  GitBranch,
  Loader2,
  RefreshCw,
  ShieldAlert,
  Users,
} from "lucide-react";
import {
  ConsistencyReportDTO,
  MergeMemoriesInput,
  app,
} from "../lib/wails";

type Props = {
  refreshKey?: number;
  currentChapter?: number;
  onGoToWiki?: (wikiID: string) => void;
  onResolved?: () => void;
};

const severityLabel: Record<string, string> = {
  ok: "正常",
  warn: "超期",
  critical: "严重超期",
};

const severityClass: Record<string, string> = {
  ok: "text-studio-muted",
  warn: "text-[rgb(var(--studio-warning-fg))]",
  critical: "text-red-500",
};

export default function ConsistencyPanel({
  refreshKey = 0,
  currentChapter = 0,
  onGoToWiki,
  onResolved,
}: Props) {
  const [report, setReport] = useState<ConsistencyReportDTO | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [resolveID, setResolveID] = useState("");
  const [resolveChapter, setResolveChapter] = useState(0);
  const [mergingSubject, setMergingSubject] = useState("");
  const [mergeKeepID, setMergeKeepID] = useState("");
  const [mergeContent, setMergeContent] = useState("");
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const r = await app().GetConsistencyReport();
      setReport(r);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setReport(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load, refreshKey]);

  const resolveForeshadow = async (id: string) => {
    try {
      await app().ResolveForeshadow(id, resolveChapter || currentChapter || 1);
      setResolveID("");
      await load();
      onResolved?.();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const archiveMemory = async (id: string) => {
    if (!confirm("归档此记忆？")) return;
    try {
      await app().ArchiveMemory(id);
      await load();
      onResolved?.();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const mergeConflict = async (subject: string) => {
    if (!mergeKeepID) return;
    setSaving(true);
    try {
      const group = report?.memory_conflicts.find((c) => c.subject === subject);
      if (!group) return;
      const archiveIDs = group.memories.filter((m) => m.id !== mergeKeepID).map((m) => m.id);
      const input: MergeMemoriesInput = {
        keep_id: mergeKeepID,
        archive_ids: archiveIDs,
        content: mergeContent.trim(),
      };
      await app().MergeMemories(input);
      setMergingSubject("");
      setMergeKeepID("");
      setMergeContent("");
      await load();
      onResolved?.();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  };

  if (loading && !report) {
    return (
      <div className="flex items-center gap-2 text-sm text-studio-muted">
        <Loader2 className="h-4 w-4 animate-spin" />
        分析一致性…
      </div>
    );
  }

  const s = report?.summary;
  const refChapter = report?.current_chapter || currentChapter;

  return (
    <div className="mx-auto min-h-0 w-full max-w-5xl flex-1 space-y-6 overflow-y-auto pb-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="flex items-center gap-2 text-lg font-semibold text-studio-text">
            <ShieldAlert className="h-5 w-5 text-studio-accent" />
            一致性仪表盘
          </h2>
          <p className="mt-1 text-sm text-studio-muted">
            参考章号：第 {refChapter} 章 · 超期伏笔阈值 ≥20 章 · 实体 stale ≥30 章
          </p>
        </div>
        <button
          type="button"
          onClick={load}
          className="inline-flex items-center gap-1.5 rounded-lg border border-studio-border px-3 py-2 text-sm text-studio-muted hover:text-studio-text"
        >
          <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
          刷新
        </button>
      </div>

      {error && <div className="studio-alert-error-compact">{error}</div>}

      {s && (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <MetricCard label="Open 伏笔" value={s.open_foreshadows} warn={s.overdue_foreshadows + s.critical_foreshadows} />
          <MetricCard label="记忆冲突" value={s.memory_conflicts} warn={s.memory_conflicts} />
          <MetricCard label="实体问题" value={s.entity_issues} warn={s.entity_issues} />
          <MetricCard
            label="待处理合计"
            value={s.total_issues}
            warn={s.total_issues}
            highlight={s.total_issues > 0}
          />
        </div>
      )}

      {s && s.total_issues === 0 && (
        <div className="flex flex-col items-center rounded-2xl border border-dashed border-studio-border py-12 text-center">
          <Check className="mb-3 h-10 w-10 text-[rgb(var(--studio-diff-add-stat))]" />
          <p className="text-sm font-medium text-studio-text">未发现一致性问题</p>
          <p className="mt-1 text-xs text-studio-muted">连载过程中可定期回来检查</p>
        </div>
      )}

      {report && report.foreshadows.length > 0 && (
        <section className="rounded-2xl border border-studio-border bg-studio-panel p-5">
          <h3 className="mb-4 flex items-center gap-2 text-sm font-medium text-studio-text">
            <GitBranch className="h-4 w-4 text-studio-muted" />
            伏笔健康
          </h3>
          <ul className="space-y-3">
            {report.foreshadows
              .slice()
              .sort((a, b) => b.gap - a.gap)
              .map((f) => (
                <li
                  key={f.id}
                  className={`rounded-xl border p-4 ${
                    f.severity === "critical"
                      ? "border-red-500/40 bg-red-500/5"
                      : f.severity === "warn"
                        ? "border-[rgb(var(--studio-warning-border))] bg-[rgb(var(--studio-warning-bg))]"
                        : "border-studio-border bg-studio-bg/40"
                  }`}
                >
                  <div className="flex flex-wrap items-start justify-between gap-3">
                    <div className="min-w-0 flex-1">
                      <p className="text-sm leading-relaxed text-studio-text">{f.description}</p>
                      <p className="mt-2 text-xs text-studio-muted">
                        埋设第 {f.planted_chapter} 章 · 已隔 {f.gap} 章 ·{" "}
                        <span className={severityClass[f.severity] ?? ""}>
                          {severityLabel[f.severity] ?? f.severity}
                        </span>
                      </p>
                    </div>
                    {f.severity !== "ok" && (
                      <div className="shrink-0">
                        {resolveID === f.id ? (
                          <div className="flex flex-wrap items-center gap-2">
                            <input
                              type="number"
                              min={1}
                              value={resolveChapter || refChapter}
                              onChange={(e) => setResolveChapter(Number(e.target.value))}
                              className="w-24 rounded border border-studio-border bg-studio-bg px-2 py-1 text-xs outline-none"
                            />
                            <button
                              type="button"
                              onClick={() => resolveForeshadow(f.id)}
                              className="rounded bg-studio-accent px-2 py-1 text-xs text-studio-on-accent"
                            >
                              确认回收
                            </button>
                            <button type="button" onClick={() => setResolveID("")} className="text-xs text-studio-muted">
                              取消
                            </button>
                          </div>
                        ) : (
                          <button
                            type="button"
                            onClick={() => {
                              setResolveID(f.id);
                              setResolveChapter(refChapter);
                            }}
                            className="text-xs text-studio-accent hover:underline"
                          >
                            标记已回收
                          </button>
                        )}
                      </div>
                    )}
                  </div>
                </li>
              ))}
          </ul>
        </section>
      )}

      {report && report.memory_conflicts.length > 0 && (
        <section className="rounded-2xl border border-studio-border bg-studio-panel p-5">
          <h3 className="mb-4 flex items-center gap-2 text-sm font-medium text-studio-text">
            <AlertTriangle className="h-4 w-4 text-[rgb(var(--studio-warning-fg))]" />
            记忆冲突
          </h3>
          <ul className="space-y-4">
            {report.memory_conflicts.map((c) => (
              <li
                key={c.subject}
                className="rounded-xl border border-[rgb(var(--studio-warning-border))] bg-[rgb(var(--studio-warning-bg))] p-4"
              >
                <p className="mb-3 text-sm font-medium">
                  「{c.subject}」· {c.count} 条活跃记忆
                </p>
                <ul className="space-y-2">
                  {c.memories.map((m) => (
                    <li key={m.id} className="rounded-lg border border-studio-border bg-studio-panel p-3">
                      <div className="flex flex-wrap items-center gap-2 text-xs text-studio-muted">
                        <span className="rounded bg-studio-border px-2 py-0.5">{m.category}</span>
                        {m.source_chapter > 0 && <span>第 {m.source_chapter} 章</span>}
                        <button
                          type="button"
                          onClick={() => archiveMemory(m.id)}
                          className="ml-auto inline-flex items-center gap-1 text-studio-muted hover:text-studio-text"
                        >
                          <Archive className="h-3 w-3" />
                          归档
                        </button>
                      </div>
                      <p className="mt-2 text-sm leading-relaxed">{m.content}</p>
                    </li>
                  ))}
                </ul>
                {mergingSubject === c.subject ? (
                  <div className="mt-4 space-y-2 rounded-lg border border-studio-border bg-studio-bg/60 p-3">
                    <p className="text-xs text-studio-muted">选择保留条目并编辑合并正文</p>
                    <select
                      value={mergeKeepID}
                      onChange={(e) => {
                        const id = e.target.value;
                        setMergeKeepID(id);
                        const mem = c.memories.find((m) => m.id === id);
                        if (mem) setMergeContent(mem.content);
                      }}
                      className="w-full rounded border border-studio-border bg-studio-panel px-2 py-1.5 text-sm"
                    >
                      <option value="">选择保留的记忆</option>
                      {c.memories.map((m) => (
                        <option key={m.id} value={m.id}>
                          [{m.category}] {m.content.slice(0, 40)}…
                        </option>
                      ))}
                    </select>
                    <textarea
                      value={mergeContent}
                      onChange={(e) => setMergeContent(e.target.value)}
                      rows={3}
                      className="w-full resize-none rounded border border-studio-border bg-studio-panel px-2 py-1.5 text-sm outline-none"
                    />
                    <div className="flex gap-2">
                      <button
                        type="button"
                        disabled={saving || !mergeKeepID}
                        onClick={() => mergeConflict(c.subject)}
                        className="rounded-lg bg-studio-accent px-3 py-1.5 text-xs text-studio-on-accent disabled:opacity-40"
                      >
                        {saving ? "合并中…" : "合并并归档其余"}
                      </button>
                      <button type="button" onClick={() => setMergingSubject("")} className="text-xs text-studio-muted">
                        取消
                      </button>
                    </div>
                  </div>
                ) : (
                  <button
                    type="button"
                    onClick={() => {
                      setMergingSubject(c.subject);
                      setMergeKeepID(c.memories[0]?.id ?? "");
                      setMergeContent(c.memories[0]?.content ?? "");
                    }}
                    className="mt-3 text-xs text-studio-accent hover:underline"
                  >
                    合并此组冲突
                  </button>
                )}
              </li>
            ))}
          </ul>
        </section>
      )}

      {report && report.entity_issues.length > 0 && (
        <section className="rounded-2xl border border-studio-border bg-studio-panel p-5">
          <h3 className="mb-4 flex items-center gap-2 text-sm font-medium text-studio-text">
            <Users className="h-4 w-4 text-studio-muted" />
            实体问题
          </h3>
          <ul className="space-y-2">
            {report.entity_issues.map((e, i) => (
              <li key={`${e.id}-${e.issue_type}-${i}`} className="rounded-lg border border-studio-border bg-studio-bg/40 px-4 py-3">
                <p className="text-sm font-medium text-studio-text">
                  {e.name || "未命名"}
                  <span className="ml-2 text-xs font-normal text-studio-muted">{e.type}</span>
                </p>
                <p className="mt-1 text-xs text-studio-muted">{e.detail}</p>
                {e.id && onGoToWiki && (
                  <button
                    type="button"
                    onClick={() => onGoToWiki(`entity:${e.id}`)}
                    className="mt-2 text-xs text-studio-accent hover:underline"
                  >
                    在设定 Wiki 查看
                  </button>
                )}
              </li>
            ))}
          </ul>
        </section>
      )}

      {report && report.cross_issues.length > 0 && (
        <section className="rounded-2xl border border-studio-border bg-studio-panel p-5">
          <h3 className="mb-2 text-sm font-medium text-studio-text">跨模块提示</h3>
          <ul className="space-y-2">
            {report.cross_issues.map((c, i) => (
              <li key={`${c.kind}-${i}`} className="rounded-lg border border-studio-border bg-studio-bg/40 px-4 py-3 text-sm">
                <p className="text-studio-text">{c.detail}</p>
                {c.memory_id && (
                  <button
                    type="button"
                    onClick={() => archiveMemory(c.memory_id!)}
                    className="mt-2 text-xs text-studio-muted hover:text-studio-text"
                  >
                    归档此记忆
                  </button>
                )}
              </li>
            ))}
          </ul>
        </section>
      )}
    </div>
  );
}

function MetricCard({
  label,
  value,
  warn = 0,
  highlight = false,
}: {
  label: string;
  value: number;
  warn?: number;
  highlight?: boolean;
}) {
  return (
    <div
      className={`rounded-xl border p-4 ${
        highlight && warn > 0
          ? "border-[rgb(var(--studio-warning-border))] bg-[rgb(var(--studio-warning-bg))]"
          : "border-studio-border bg-studio-panel"
      }`}
    >
      <p className="text-[10px] uppercase tracking-wide text-studio-muted">{label}</p>
      <p className="mt-1 text-2xl font-semibold tabular-nums text-studio-text">{value}</p>
      {warn > 0 && label !== "待处理合计" && (
        <p className="mt-1 text-[11px] text-[rgb(var(--studio-warning-fg))]">{warn} 需关注</p>
      )}
    </div>
  );
}
