import { useEffect, useState } from "react";
import { Loader2, Plus } from "lucide-react";
import { SettingCategory } from "../lib/wikiCategories";
import { app } from "../lib/wails";

type Props = {
  open: boolean;
  category: SettingCategory;
  onClose: () => void;
  onCreated: (id: string) => void;
};

const templateOptions: Record<
  SettingCategory,
  { id: string; label: string }[]
> = {
  角色: [
    { id: "character", label: "角色卡" },
    { id: "villain", label: "反派" },
    { id: "blank", label: "空白" },
  ],
  背景: [{ id: "default", label: "世界观条目" }],
  势力: [{ id: "default", label: "势力条目" }],
  地点: [{ id: "default", label: "地点条目" }],
  物品: [{ id: "default", label: "物品条目" }],
  其他: [{ id: "blank", label: "空白" }],
};

export default function CreateSettingDialog({
  open,
  category,
  onClose,
  onCreated,
}: Props) {
  const [title, setTitle] = useState("");
  const [templateKind, setTemplateKind] = useState("character");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  const templates = templateOptions[category];

  useEffect(() => {
    if (!open) return;
    setTitle("");
    setError("");
    setTemplateKind(templates[0]?.id ?? "default");
  }, [open, category, templates]);

  if (!open) return null;

  const submit = async () => {
    const name = title.trim();
    if (!name) {
      setError("请填写名称");
      return;
    }
    setSaving(true);
    setError("");
    try {
      const created = await app().CreateWikiSetting({
        category,
        title: name,
        template_kind: templateKind,
      });
      onCreated(created.id);
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
        className="w-full max-w-md rounded-xl border border-studio-border bg-studio-panel shadow-card"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="border-b border-studio-border px-5 py-4">
          <div className="flex items-center gap-2">
            <Plus className="h-4 w-4 text-studio-accent" />
            <h2 className="text-base font-medium">新建{category}设定</h2>
          </div>
          <p className="mt-1 text-xs text-studio-muted">
            保存至设定集/{category === "背景" ? "世界" : category}/，可在右侧直接编辑
          </p>
        </div>

        <div className="space-y-4 px-5 py-4">
          <label className="block">
            <span className="text-xs font-medium text-studio-muted">名称</span>
            <input
              autoFocus
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") void submit();
              }}
              placeholder={category === "角色" ? "如：配角-李师兄" : "如：青云宗"}
              className="mt-1.5 w-full rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none focus:border-studio-accent/50"
            />
          </label>

          {templates.length > 1 && (
            <label className="block">
              <span className="text-xs font-medium text-studio-muted">模板</span>
              <select
                value={templateKind}
                onChange={(e) => setTemplateKind(e.target.value)}
                className="mt-1.5 w-full rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none focus:border-studio-accent/50"
              >
                {templates.map((t) => (
                  <option key={t.id} value={t.id}>
                    {t.label}
                  </option>
                ))}
              </select>
            </label>
          )}

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
            创建并编辑
          </button>
        </div>
      </div>
    </div>
  );
}
