import { useCallback, useEffect, useState } from "react";
import { Loader2 } from "lucide-react";
import { OutlineMatrixDTO, app } from "../lib/wails";

type Props = {
  volume: number;
  refreshKey?: number;
  selected: number[];
  onChange: (chapters: number[]) => void;
};

export default function PolishChapterPicker({
  volume,
  refreshKey = 0,
  selected,
  onChange,
}: Props) {
  const [matrix, setMatrix] = useState<OutlineMatrixDTO | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await app().GetOutlineChapterMatrix(volume);
      setMatrix(data);
    } catch {
      setMatrix(null);
    } finally {
      setLoading(false);
    }
  }, [volume]);

  useEffect(() => {
    void load();
  }, [load, refreshKey]);

  const written = (matrix?.rows ?? []).filter((r) => r.has_body);

  const toggle = (chapter: number) => {
    onChange(
      selected.includes(chapter)
        ? selected.filter((n) => n !== chapter)
        : [...selected, chapter].sort((a, b) => a - b),
    );
  };

  const selectAll = () => onChange(written.map((r) => r.chapter));
  const clearAll = () => onChange([]);

  if (loading) {
    return (
      <div className="flex items-center gap-2 py-2 text-xs text-studio-muted">
        <Loader2 className="h-3.5 w-3.5 animate-spin" />
        加载章节列表…
      </div>
    );
  }

  if (written.length === 0) {
    return (
      <p className="rounded-lg border border-studio-border bg-studio-bg/40 px-3 py-2 text-xs text-studio-muted">
        当前卷尚无已写正文，无法批量润色。
      </p>
    );
  }

  return (
    <div className="space-y-2 rounded-lg border border-studio-border bg-studio-bg/40 p-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-xs text-studio-muted">
          勾选要润色的章节（仅已写正文，共 {written.length} 章）
        </p>
        <div className="flex gap-2 text-[11px]">
          <button type="button" onClick={selectAll} className="text-studio-accent hover:underline">
            全选
          </button>
          <button type="button" onClick={clearAll} className="text-studio-muted hover:underline">
            清空
          </button>
        </div>
      </div>
      <ul className="max-h-40 space-y-1 overflow-y-auto">
        {written.map((row) => (
          <li key={row.chapter}>
            <label className="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-xs hover:bg-studio-panel">
              <input
                type="checkbox"
                checked={selected.includes(row.chapter)}
                onChange={() => toggle(row.chapter)}
              />
              <span className="tabular-nums">第 {row.chapter} 章</span>
              <span className="min-w-0 flex-1 truncate text-studio-muted">
                {row.title || "无标题"}
              </span>
              <span className="shrink-0 tabular-nums text-studio-muted/80">{row.word_count} 字</span>
            </label>
          </li>
        ))}
      </ul>
    </div>
  );
}
