import { useEffect, useRef, useState } from "react";
import { AlertTriangle } from "lucide-react";
import { UnsavedChoice, setUnsavedPromptHandler } from "../lib/unsavedGuard";

export default function UnsavedChangesDialog() {
  const [open, setOpen] = useState(false);
  const resolveRef = useRef<((choice: UnsavedChoice) => void) | null>(null);

  useEffect(() => {
    setUnsavedPromptHandler((resolve) => {
      resolveRef.current = resolve;
      setOpen(true);
    });
    return () => setUnsavedPromptHandler(null);
  }, []);

  const choose = (choice: UnsavedChoice) => {
    setOpen(false);
    const resolve = resolveRef.current;
    resolveRef.current = null;
    resolve?.(choice);
  };

  if (!open) return null;

  return (
    <div className="studio-modal-overlay fixed inset-0 z-[60] flex items-center justify-center p-4">
      <div
        className="w-full max-w-md rounded-xl border border-studio-border bg-studio-panel shadow-card"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-start gap-3 border-b border-studio-border px-5 py-4">
          <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0 text-[rgb(var(--studio-warning-fg))]" />
          <div>
            <h2 className="text-base font-medium">有未保存的更改</h2>
            <p className="mt-1 text-sm text-studio-muted">离开前是否保存当前编辑内容？</p>
          </div>
        </div>
        <div className="flex flex-wrap justify-end gap-2 px-5 py-4">
          <button
            type="button"
            onClick={() => choose("cancel")}
            className="rounded-lg border border-studio-border px-4 py-2 text-sm hover:bg-studio-bg"
          >
            留在此页
          </button>
          <button
            type="button"
            onClick={() => choose("discard")}
            className="rounded-lg border border-studio-border px-4 py-2 text-sm text-studio-muted hover:bg-studio-bg hover:text-studio-text"
          >
            不保存
          </button>
          <button
            type="button"
            onClick={() => choose("save")}
            className="rounded-lg bg-studio-accent px-4 py-2 text-sm font-medium text-studio-on-accent hover:brightness-110"
          >
            保存并离开
          </button>
        </div>
      </div>
    </div>
  );
}
