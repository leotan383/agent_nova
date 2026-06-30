import { useCallback, useEffect, useState } from "react";
import { AlertCircle, CheckCircle2, Loader2, RefreshCw } from "lucide-react";
import { GateCheckDTO, WriteGateDTO, app } from "../lib/wails";

type Props = {
  chapter: number;
  volume: number;
};

export default function WriteGatePanel({ chapter, volume }: Props) {
  const [gate, setGate] = useState<WriteGateDTO | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    if (chapter <= 0) return;
    setLoading(true);
    setError("");
    try {
      const g = await app().GetWriteGate(chapter, volume);
      setGate(g);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setGate(null);
    } finally {
      setLoading(false);
    }
  }, [chapter, volume]);

  useEffect(() => {
    load();
  }, [load]);

  const blockingFails = gate?.checks.filter((c) => c.blocking && !c.ok) ?? [];
  const warnings = gate?.checks.filter((c) => !c.blocking && !c.ok) ?? [];

  return (
    <div className="rounded-xl border border-studio-border bg-studio-panel p-4">
      <div className="mb-3 flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <h3 className="text-sm font-medium">写前检查</h3>
          {loading ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin text-studio-muted" />
          ) : gate?.ok ? (
            <span className="inline-flex items-center gap-1 rounded-full bg-[rgb(var(--studio-diff-add-bg))] px-2 py-0.5 text-xs text-[rgb(var(--studio-diff-add-stat))]">
              <CheckCircle2 className="h-3 w-3" />
              可开写
            </span>
          ) : gate ? (
            <span className="inline-flex items-center gap-1 rounded-full bg-[rgb(var(--studio-warning-bg))] px-2 py-0.5 text-xs text-[rgb(var(--studio-warning-fg))]">
              <AlertCircle className="h-3 w-3" />
              需处理 {blockingFails.length} 项
            </span>
          ) : null}
        </div>
        <button
          type="button"
          onClick={load}
          disabled={loading}
          className="inline-flex items-center gap-1 rounded-lg px-2 py-1 text-xs text-studio-muted hover:bg-studio-bg hover:text-studio-text"
        >
          <RefreshCw className="h-3 w-3" />
          刷新
        </button>
      </div>

      {error && <div className="mb-3 studio-alert-error-compact">{error}</div>}

      {gate && (
        <ul className="space-y-2">
          {gate.checks.map((c) => (
            <GateCheckRow key={c.key} check={c} />
          ))}
        </ul>
      )}

      {warnings.length > 0 && (
        <p className="mt-3 text-xs text-studio-muted">
          黄色提示项不阻断写章，但可能影响一致性。
        </p>
      )}
    </div>
  );
}

function GateCheckRow({ check }: { check: GateCheckDTO }) {
  const ok = check.ok;
  const warn = !check.blocking && !ok;
  return (
    <li
      className={`flex items-start gap-2 rounded-lg border px-3 py-2 text-sm ${
        ok
          ? "border-studio-border/80 bg-studio-bg/50"
          : warn
            ? "border-[rgb(var(--studio-warning-border))] bg-[rgb(var(--studio-warning-bg)/0.5)]"
            : "border-[rgb(var(--studio-danger-border))] bg-[rgb(var(--studio-danger-bg)/0.5)]"
      }`}
    >
      {ok ? (
        <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0 text-[rgb(var(--studio-diff-add-stat))]" />
      ) : (
        <AlertCircle
          className={`mt-0.5 h-4 w-4 shrink-0 ${warn ? "text-[rgb(var(--studio-warning-fg))]" : "text-[rgb(var(--studio-danger-fg))]"}`}
        />
      )}
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-2">
          <span className="font-medium">{check.label}</span>
          {!check.blocking && (
            <span className="rounded bg-studio-border px-1.5 py-0.5 text-[10px] text-studio-muted">提示</span>
          )}
        </div>
        <p className="mt-0.5 text-xs text-studio-muted">{check.detail}</p>
      </div>
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
