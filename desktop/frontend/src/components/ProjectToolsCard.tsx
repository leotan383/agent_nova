import { useCallback, useEffect, useState } from "react";
import {
  ArchiveRestore,
  Brain,
  Check,
  HardDriveDownload,
  Loader2,
  Stethoscope,
} from "lucide-react";
import {
  BackupItemDTO,
  DoctorReportDTO,
  PreflightDTO,
  app,
} from "../lib/wails";

type Props = {
  refreshKey?: number;
  onRefresh?: () => void;
};

export default function ProjectToolsCard({ refreshKey = 0, onRefresh }: Props) {
  const [backups, setBackups] = useState<BackupItemDTO[]>([]);
  const [loadingBackups, setLoadingBackups] = useState(true);
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");
  const [doctor, setDoctor] = useState<DoctorReportDTO | null>(null);
  const [preflight, setPreflight] = useState<PreflightDTO | null>(null);
  const [bootstrapMsg, setBootstrapMsg] = useState("");

  const loadBackups = useCallback(async () => {
    setLoadingBackups(true);
    try {
      const list = await app().ListProjectBackups();
      setBackups(list ?? []);
    } catch {
      setBackups([]);
    } finally {
      setLoadingBackups(false);
    }
  }, []);

  useEffect(() => {
    loadBackups();
  }, [loadBackups, refreshKey]);

  const runDoctor = async (deep: boolean) => {
    setBusy(deep ? "doctor-deep" : "doctor");
    setError("");
    try {
      const rep = await app().RunProjectDoctor(deep);
      setDoctor(rep);
      if (!deep) {
        const pf = await app().RunPreflight();
        setPreflight(pf);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy("");
    }
  };

  const createBackup = async () => {
    setBusy("backup");
    setError("");
    try {
      await app().CreateProjectBackup("manual");
      await loadBackups();
      onRefresh?.();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy("");
    }
  };

  const restoreBackup = async (name: string) => {
    if (!confirm(`确定从备份「${name}」恢复？将覆盖当前正文、设定与大纲。`)) return;
    setBusy(`restore-${name}`);
    setError("");
    try {
      await app().RestoreProjectBackup(name);
      onRefresh?.();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy("");
    }
  };

  const bootstrap = async () => {
    setBusy("bootstrap");
    setError("");
    setBootstrapMsg("");
    try {
      const res = await app().BootstrapMemories();
      setBootstrapMsg(`已新增 ${res.inserted_count} 条 world 记忆`);
      onRefresh?.();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy("");
    }
  };

  return (
    <div className="rounded-xl border border-studio-border bg-studio-panel p-4">
      <h3 className="mb-3 text-xs font-medium uppercase tracking-wide text-studio-muted">项目工具</h3>
      {error && <div className="mb-3 studio-alert-error-compact text-xs">{error}</div>}

      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          disabled={!!busy}
          onClick={() => runDoctor(false)}
          className="inline-flex items-center gap-1.5 rounded-lg border border-studio-border px-3 py-2 text-xs text-studio-text hover:bg-studio-bg disabled:opacity-50"
        >
          {busy === "doctor" ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Stethoscope className="h-3.5 w-3.5" />}
          项目体检
        </button>
        <button
          type="button"
          disabled={!!busy}
          onClick={createBackup}
          className="inline-flex items-center gap-1.5 rounded-lg border border-studio-border px-3 py-2 text-xs text-studio-text hover:bg-studio-bg disabled:opacity-50"
        >
          {busy === "backup" ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <HardDriveDownload className="h-3.5 w-3.5" />}
          立即备份
        </button>
        <button
          type="button"
          disabled={!!busy}
          onClick={bootstrap}
          className="inline-flex items-center gap-1.5 rounded-lg border border-studio-border px-3 py-2 text-xs text-studio-text hover:bg-studio-bg disabled:opacity-50"
        >
          {busy === "bootstrap" ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Brain className="h-3.5 w-3.5" />}
          设定→记忆
        </button>
      </div>

      {bootstrapMsg && (
        <p className="mt-2 flex items-center gap-1 text-xs text-[rgb(var(--studio-diff-add-stat))]">
          <Check className="h-3.5 w-3.5" />
          {bootstrapMsg}
        </p>
      )}

      {(doctor || preflight) && (
        <div className="mt-4 space-y-2 rounded-lg border border-studio-border/60 bg-studio-bg/40 p-3">
          {preflight && (
            <p className={`text-xs font-medium ${preflight.ok ? "text-[rgb(var(--studio-diff-add-stat))]" : "text-[rgb(var(--studio-warning-fg))]"}`}>
              预检：{preflight.ok ? "通过" : "未通过"}
            </p>
          )}
          {doctor?.findings.map((f, i) => (
            <div key={i} className="text-xs">
              <span
                className={
                  f.level === "error"
                    ? "text-red-500"
                    : f.level === "warn"
                      ? "text-[rgb(var(--studio-warning-fg))]"
                      : "text-studio-muted"
                }
              >
                [{f.level}] {f.message}
              </span>
              {f.fix && <span className="ml-1 text-studio-muted">→ {f.fix}</span>}
            </div>
          ))}
          <button
            type="button"
            disabled={!!busy}
            onClick={() => runDoctor(true)}
            className="text-[11px] text-studio-accent hover:underline disabled:opacity-50"
          >
            深度检查
          </button>
        </div>
      )}

      <div className="mt-4">
        <p className="mb-2 text-[10px] uppercase tracking-wide text-studio-muted">最近备份</p>
        {loadingBackups ? (
          <Loader2 className="h-4 w-4 animate-spin text-studio-muted" />
        ) : backups.length === 0 ? (
          <p className="text-xs text-studio-muted">暂无备份</p>
        ) : (
          <ul className="max-h-28 space-y-1 overflow-y-auto">
            {backups.slice(0, 8).map((b) => (
              <li key={b.name} className="flex items-center justify-between gap-2 rounded-md bg-studio-bg/50 px-2 py-1.5">
                <span className="truncate text-xs text-studio-text">{b.name}</span>
                <button
                  type="button"
                  disabled={!!busy}
                  onClick={() => restoreBackup(b.name)}
                  className="inline-flex shrink-0 items-center gap-1 text-[10px] text-studio-muted hover:text-studio-text disabled:opacity-50"
                >
                  <ArchiveRestore className="h-3 w-3" />
                  恢复
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
