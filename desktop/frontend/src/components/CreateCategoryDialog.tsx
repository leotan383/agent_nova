import { useEffect, useState } from "react";
import { LayoutGrid, Loader2, Pencil } from "lucide-react";
import { app } from "../lib/wails";

type Props = {
  open: boolean;
  mode?: "create" | "rename";
  categoryId?: string;
  onClose: () => void;
  onCreated: (categoryId: string) => void;
  onRenamed?: (categoryId: string) => void;
};

export default function CreateCategoryDialog({
  open,
  mode = "create",
  categoryId = "",
  onClose,
  onCreated,
  onRenamed,
}: Props) {
  const [name, setName] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const isRename = mode === "rename";

  useEffect(() => {
    if (!open) return;
    setName(isRename ? categoryId : "");
    setError("");
  }, [open, isRename, categoryId]);

  if (!open) return null;

  const submit = async () => {
    const trimmed = name.trim();
    if (!trimmed) {
      setError("请填写分类名称");
      return;
    }
    setSaving(true);
    setError("");
    try {
      if (isRename) {
        const renamed = await app().RenameSettingCategory(categoryId, trimmed);
        onRenamed?.(renamed.id);
      } else {
        const created = await app().CreateSettingCategory(trimmed);
        onCreated(created.id);
      }
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div
      className="studio-modal-overlay fixed inset-0 z-[60] flex items-center justify-center p-4"
      onClick={onClose}
    >
      <div
        className="w-full max-w-sm rounded-xl border border-studio-border bg-studio-panel shadow-card"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="border-b border-studio-border px-5 py-4">
          <div className="flex items-center gap-2">
            {isRename ? (
              <Pencil className="h-4 w-4 text-studio-accent" />
            ) : (
              <LayoutGrid className="h-4 w-4 text-studio-accent" />
            )}
            <h2 className="text-base font-medium">
              {isRename ? "重命名设定分类" : "新建设定分类"}
            </h2>
          </div>
          <p className="mt-1 text-xs text-studio-muted">
            {isRename
              ? "将同步重命名设定集下的对应文件夹"
              : "将在设定集下创建独立文件夹，例如「功法」「种族」"}
          </p>
        </div>

        <div className="space-y-3 px-5 py-4">
          <label className="block">
            <span className="text-xs font-medium text-studio-muted">分类名称</span>
            <input
              autoFocus
              value={name}
              onChange={(e) => setName(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") void submit();
              }}
              placeholder="如：功法、种族、术语"
              className="mt-1.5 w-full rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none focus:border-studio-accent/50"
            />
          </label>
          {error && <p className="text-xs text-[rgb(var(--studio-danger-fg))]">{error}</p>}
        </div>

        <div className="flex justify-end gap-2 border-t border-studio-border px-5 py-4">
          <button
            type="button"
            onClick={onClose}
            disabled={saving}
            className="rounded-lg border border-studio-border px-4 py-2 text-sm hover:bg-studio-bg disabled:opacity-50"
          >
            取消
          </button>
          <button
            type="button"
            onClick={() => void submit()}
            disabled={saving}
            className="inline-flex items-center gap-1.5 rounded-lg bg-studio-accent px-4 py-2 text-sm font-medium text-studio-on-accent hover:brightness-110 disabled:opacity-50"
          >
            {saving && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
            {isRename ? "保存" : "创建分类"}
          </button>
        </div>
      </div>
    </div>
  );
}
