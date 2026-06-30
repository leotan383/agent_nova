import { useCallback, useEffect, useState } from "react";
import { AlertCircle, CheckCircle2, Loader2, RefreshCw } from "lucide-react";
import { GateCheckDTO, WriteGateDTO, app } from "../lib/wails";

type Props = {
  chapter: number;
  volume: number;
  compact?: boolean;
  onFix?: (checkKey: string) => void;
  onReadyChange?: (ready: boolean) => void;
};

export default function WriteGatePanel({
  chapter,
  volume,
  compact = false,
  onFix,
  onReadyChange,
}: Props) {
  const [gate, setGate] = useState<WriteGateDTO | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [expanded, setExpanded] = useState(false);

  const load = useCallback(async () => {
    if (chapter <= 0) return;
    setLoading(true);
    setError("");
    try {
      const g = await app().GetWriteGate(chapter, volume);
      setGate(g);
      onReadyChange?.(g.ok);
      if (!g.ok) setExpanded(true);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setGate(null);
      onReadyChange?.(false);
    } finally {
      setLoading(false);
    }
  }, [chapter, volume, onReadyChange]);

  useEffect(() => {
    setExpanded(false);
    load();
  }, [load]);

  const blockingFails = gate?.checks.filter((c) => c.blocking && !c.ok) ?? [];
  const warnings = gate?.checks.filter((c) => !c.blocking && !c.ok) ?? [];
  const showCompact = compact && gate?.ok && !expanded;

  if (showCompact) {
    return (
      <div className="flex items-center justify-between gap-2 rounded-lg border border-[rgb(var(--studio-diff-add-stat)/0.25)] bg-[rgb(var(--studio-diff-add-bg)/0.6)] px-3 py-2">
        <span className="inline-flex items-center gap-1.5 text-xs text-[rgb(var(--studio-diff-add-stat))]">
          <CheckCircle2 className="h-3.5 w-3.5" />
          写前检查已通过
        </span>
        <button
          type="button"
          onClick={() => setExpanded(true)}
          className="text-[10px] text-studio-muted hover:text-studio-text"
        >
          详情
        </button>
      </div>
    );
  }

  return (
    <div className="rounded-xl border border-studio-border bg-studio-panel p-3">
      <div className="mb-2 flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <h3 className="text-xs font-medium uppercase tracking-wide text-studio-muted">写前检查</h3>
          {loading ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin text-studio-muted" />
          ) : gate?.ok ? (
            <span className="inline-flex items-center gap-1 rounded-full bg-[rgb(var(--studio-diff-add-bg))] px-2 py-0.5 text-[10px] text-[rgb(var(--studio-diff-add-stat))]">
              <CheckCircle2 className="h-3 w-3" />
              可开写
            </span>
          ) : gate ? (
            <span className="inline-flex items-center gap-1 rounded-full bg-[rgb(var(--studio-warning-bg))] px-2 py-0.5 text-[10px] text-[rgb(var(--studio-warning-fg))]">
              <AlertCircle className="h-3 w-3" />
              {blockingFails.length} 项待处理
            </span>
          ) : null}
        </div>
        <div className="flex items-center gap-1">
          {compact && gate?.ok && (
            <button
              type="button"
              onClick={() => setExpanded(false)}
              className="rounded px-1.5 py-0.5 text-[10px] text-studio-muted hover:bg-studio-bg"
            >
              收起
            </button>
          )}
          <button
            type="button"
            onClick={load}
            disabled={loading}
            className="rounded p-1 text-studio-muted hover:text-studio-text"
            title="刷新"
          >
            <RefreshCw className={`h-3.5 w-3.5 ${loading ? "animate-spin" : ""}`} />
          </button>
        </div>
      </div>

      {error && <div className="mb-2 studio-alert-error-compact">{error}</div>}

      {gate && (
        <ul className="space-y-1.5">
          {gate.checks.map((c) => (
            <GateCheckRow key={c.key} check={c} onFix={onFix} />
          ))}
        </ul>
      )}

      {warnings.length > 0 && (
        <p className="mt-2 text-[10px] text-studio-muted">黄色项不阻断写章，但可能影响一致性。</p>
      )}
    </div>
  );
}

function fixLabel(key: string): string | null {
  switch (key) {
    case "volume_outline":
    case "chapter_outline":
      return "去规划";
    case "prev_summary":
      return "看上章";
    case "index":
      return "重建索引";
    default:
      return null;
  }
}

function GateCheckRow({
  check,
  onFix,
}: {
  check: GateCheckDTO;
  onFix?: (key: string) => void;
}) {
  const ok = check.ok;
  const warn = !check.blocking && !ok;
  const fix = !ok ? fixLabel(check.key) : null;

  return (
    <li
      className={`flex items-start gap-2 rounded-lg border px-2.5 py-2 text-xs ${
        ok
          ? "border-studio-border/60 bg-studio-bg/40"
          : warn
            ? "border-[rgb(var(--studio-warning-border))] bg-[rgb(var(--studio-warning-bg)/0.4)]"
            : "border-[rgb(var(--studio-danger-border))] bg-[rgb(var(--studio-danger-bg)/0.4)]"
      }`}
    >
      {ok ? (
        <CheckCircle2 className="mt-0.5 h-3.5 w-3.5 shrink-0 text-[rgb(var(--studio-diff-add-stat))]" />
      ) : (
        <AlertCircle
          className={`mt-0.5 h-3.5 w-3.5 shrink-0 ${warn ? "text-[rgb(var(--studio-warning-fg))]" : "text-[rgb(var(--studio-danger-fg))]"}`}
        />
      )}
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-1.5">
          <span className="font-medium">{check.label}</span>
          {!check.blocking && (
            <span className="rounded bg-studio-border px-1 py-0.5 text-[9px] text-studio-muted">提示</span>
          )}
        </div>
        <p className="mt-0.5 text-[10px] leading-relaxed text-studio-muted">{check.detail}</p>
      </div>
      {fix && onFix && (
        <button
          type="button"
          onClick={() => onFix(check.key)}
          className="shrink-0 rounded-md border border-studio-border px-2 py-0.5 text-[10px] hover:bg-studio-bg"
        >
          {fix}
        </button>
      )}
    </li>
  );
}

export function useWriteGateReady(chapter: number, volume: number) {
  const [ready, setReady] = useState(false);
  useEffect(() => {
    if (chapter <= 0) {
      setReady(false);
      return;
    }
    app()
      .GetWriteGate(chapter, volume)
      .then((g) => setReady(g.ok))
      .catch(() => setReady(false));
  }, [chapter, volume]);
  return ready;
}
