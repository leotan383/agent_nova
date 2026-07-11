import { useState } from "react";
import { Loader2 } from "lucide-react";
import { app } from "../lib/wails";

type Props = {
  onSaved: () => void;
  onCancel: () => void;
};

export default function InspirationQuickCapture({ onSaved, onCancel }: Props) {
  const [spark, setSpark] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  const save = async () => {
    if (!spark.trim()) {
      setError("写点什么吧，一句话也行");
      return;
    }
    setSaving(true);
    setError("");
    try {
      await app().CreateInspiration({ spark: spark.trim() });
      onSaved();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-4">
      <textarea
        value={spark}
        onChange={(e) => setSpark(e.target.value)}
        placeholder="随便写，一句话也行…"
        rows={6}
        autoFocus
        onKeyDown={(e) => {
          if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
            e.preventDefault();
            void save();
          }
        }}
        className="w-full resize-none rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none focus:border-studio-accent"
      />
      <p className="text-[10px] text-studio-muted">⌘/Ctrl + Enter 保存 · 标题、标签等可之后点卡片再编辑</p>
      {error && <div className="studio-alert-error-compact">{error}</div>}
      <div className="flex justify-end gap-3">
        <button type="button" onClick={onCancel} className="px-4 py-2 text-sm text-studio-muted">
          取消
        </button>
        <button
          type="button"
          onClick={() => void save()}
          disabled={saving || !spark.trim()}
          className="inline-flex items-center gap-1 rounded-lg bg-studio-accent px-4 py-2 text-sm font-medium text-studio-on-accent disabled:opacity-40"
        >
          {saving && <Loader2 className="h-4 w-4 animate-spin" />}
          保存
        </button>
      </div>
    </div>
  );
}
