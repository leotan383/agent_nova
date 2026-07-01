import { AlertTriangle } from "lucide-react";

type Props = {
  open: boolean;
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  destructive?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
};

export default function ConfirmDialog({
  open,
  title,
  message,
  confirmLabel = "确认",
  cancelLabel = "取消",
  destructive = false,
  onConfirm,
  onCancel,
}: Props) {
  if (!open) return null;

  return (
    <div className="studio-modal-overlay fixed inset-0 z-[60] flex items-center justify-center p-4">
      <div
        className="w-full max-w-md rounded-xl border border-studio-border bg-studio-panel shadow-card"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-start gap-3 border-b border-studio-border px-5 py-4">
          <AlertTriangle
            className={`mt-0.5 h-5 w-5 shrink-0 ${
              destructive ? "text-[rgb(var(--studio-danger-fg))]" : "text-[rgb(var(--studio-warning-fg))]"
            }`}
          />
          <div>
            <h2 className="text-base font-medium">{title}</h2>
            <p className="mt-1 text-sm leading-relaxed text-studio-muted">{message}</p>
          </div>
        </div>
        <div className="flex justify-end gap-2 px-5 py-4">
          <button
            type="button"
            onClick={onCancel}
            className="rounded-lg border border-studio-border px-4 py-2 text-sm hover:bg-studio-bg"
          >
            {cancelLabel}
          </button>
          <button
            type="button"
            onClick={onConfirm}
            className={`rounded-lg px-4 py-2 text-sm font-medium ${
              destructive
                ? "bg-[rgb(var(--studio-danger-bg))] text-[rgb(var(--studio-danger-fg))] hover:brightness-95"
                : "bg-studio-accent text-studio-on-accent hover:brightness-110"
            }`}
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
