import { useEffect, useState } from "react";
import { Lightbulb, Loader2, X } from "lucide-react";
import { InspirationCard, InspirationPrefillDTO, app } from "../lib/wails";
import { inspirationStatusLabel } from "../lib/inspirationUtils";

type Props = {
  selectedId: string;
  onSelect: (prefill: InspirationPrefillDTO | null) => void;
};

export default function InspirationPicker({ selectedId, onSelect }: Props) {
  const [items, setItems] = useState<InspirationCard[]>([]);
  const [loading, setLoading] = useState(true);
  const [open, setOpen] = useState(false);

  const load = async () => {
    setLoading(true);
    try {
      const list = await app().ListInspirations({ include_archived: false });
      setItems((list ?? []).filter((i) => i.status !== "used"));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load();
  }, []);

  const pick = async (id: string) => {
    const prefill = await app().GetInspirationPrefill(id);
    onSelect(prefill);
    setOpen(false);
  };

  const clear = () => {
    onSelect(null);
    setOpen(false);
  };

  const selected = items.find((i) => i.id === selectedId);

  return (
    <div className="rounded-xl border border-studio-border bg-studio-bg/60 p-3">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="text-xs font-medium text-studio-muted">从灵感开始（可选）</p>
          {selectedId && selected ? (
            <p className="mt-1 truncate text-sm">
              <Lightbulb className="mr-1 inline h-3.5 w-3.5 text-studio-accent" />
              {selected.title}
              <span className="ml-2 text-xs text-studio-muted">
                {inspirationStatusLabel[selected.status]}
              </span>
            </p>
          ) : (
            <p className="mt-1 text-sm text-studio-muted">选择灵感库中的条目，自动预填创书表单</p>
          )}
        </div>
        <div className="flex shrink-0 gap-1">
          {selectedId && (
            <button
              type="button"
              onClick={clear}
              className="rounded-lg p-1.5 text-studio-muted hover:bg-studio-panel hover:text-studio-text"
              title="清除"
            >
              <X className="h-4 w-4" />
            </button>
          )}
          <button
            type="button"
            onClick={() => setOpen((v) => !v)}
            className="rounded-lg border border-studio-border px-3 py-1.5 text-xs hover:bg-studio-panel"
          >
            {selectedId ? "更换" : "选择灵感"}
          </button>
        </div>
      </div>

      {open && (
        <div className="mt-3 max-h-48 space-y-1 overflow-y-auto rounded-lg border border-studio-border bg-studio-panel p-1">
          {loading ? (
            <div className="flex items-center justify-center py-6 text-xs text-studio-muted">
              <Loader2 className="mr-1 h-3 w-3 animate-spin" />
              加载中…
            </div>
          ) : items.length === 0 ? (
            <p className="px-2 py-4 text-center text-xs text-studio-muted">灵感库还是空的，先去记录一个吧</p>
          ) : (
            items.map((item) => (
              <button
                key={item.id}
                type="button"
                onClick={() => void pick(item.id)}
                className={`block w-full rounded-md px-2 py-2 text-left text-sm hover:bg-studio-bg ${
                  item.id === selectedId ? "bg-studio-accent/10 text-studio-accent" : ""
                }`}
              >
                <span className="font-medium">{item.title}</span>
                <span className="mt-0.5 block truncate text-xs text-studio-muted">{item.spark_preview}</span>
              </button>
            ))
          )}
        </div>
      )}
    </div>
  );
}
