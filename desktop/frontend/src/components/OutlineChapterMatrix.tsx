import { useCallback, useEffect, useState } from "react";
import { ExternalLink, Loader2, RefreshCw } from "lucide-react";
import { OutlineMatrixDTO, app } from "../lib/wails";

const matchLabels: Record<string, string> = {
  matched: "已完成",
  unwritten: "未写",
  deviated: "偏离",
  abandoned: "废弃",
};

const matchClass: Record<string, string> = {
  matched: "text-[rgb(var(--studio-diff-add-stat))]",
  unwritten: "text-studio-muted",
  deviated: "text-amber-600 dark:text-amber-400",
  abandoned: "text-studio-muted/60 line-through",
};

type Props = {
  volume: number;
  refreshKey?: number;
  onOpenChapter?: (chapter: number) => void;
  onStartReplan?: () => void;
  onInsertAfter?: (after: number) => void;
  selectedChapters?: number[];
  onToggleChapter?: (chapter: number) => void;
};

export default function OutlineChapterMatrix({
  volume,
  refreshKey = 0,
  onOpenChapter,
  onStartReplan,
  onInsertAfter,
  selectedChapters,
  onToggleChapter,
}: Props) {
  const [matrix, setMatrix] = useState<OutlineMatrixDTO | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const data = await app().GetOutlineChapterMatrix(volume);
      setMatrix(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setMatrix(null);
    } finally {
      setLoading(false);
    }
  }, [volume]);

  useEffect(() => {
    void load();
  }, [load, refreshKey]);

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12 text-sm text-studio-muted">
        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
        加载对照矩阵…
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg border border-[rgb(var(--studio-danger-border))] bg-[rgb(var(--studio-danger-bg))] px-3 py-2 text-sm text-[rgb(var(--studio-danger-fg))]">
        {error}
      </div>
    );
  }

  if (!matrix) return null;

  const s = matrix.summary;
  const selectable = !!onToggleChapter;

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-hidden">
      <div className="flex shrink-0 flex-wrap items-center gap-2 text-xs text-studio-muted">
        <span>
          {s.written}/{s.total_in_outline || matrix.rows.length} 已写
        </span>
        {s.deviated > 0 && <span className="text-amber-600">· {s.deviated} 偏离</span>}
        {s.unwritten > 0 && <span>· {s.unwritten} 未写</span>}
        <button
          type="button"
          onClick={() => void load()}
          className="ml-auto inline-flex items-center gap-1 rounded-md px-2 py-1 hover:bg-studio-bg"
        >
          <RefreshCw className="h-3 w-3" />
          刷新
        </button>
        {s.deviated > 0 && onStartReplan && (
          <button
            type="button"
            onClick={onStartReplan}
            className="rounded-md bg-studio-accent/15 px-2 py-1 text-studio-accent hover:bg-studio-accent/25"
          >
            发起 Replan
          </button>
        )}
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto rounded-lg border border-studio-border">
        {matrix.rows.length === 0 ? (
          <div className="flex flex-col items-center justify-center gap-2 px-6 py-16 text-center text-sm text-studio-muted">
            <p>本卷尚无章纲条目，对照列表为空。</p>
            <p className="text-xs">
              请先在「卷纲编辑」生成或编写第 {volume} 卷卷纲；若正文写在其他卷，请切换卷号查看。
            </p>
          </div>
        ) : (
        <table className="w-full min-w-[520px] text-left text-xs">
          <thead className="sticky top-0 bg-studio-panel text-studio-muted">
            <tr className="border-b border-studio-border">
              {selectable && <th className="w-8 px-2 py-2" />}
              <th className="px-3 py-2 font-medium">章号</th>
              <th className="px-3 py-2 font-medium">卷纲</th>
              <th className="px-3 py-2 font-medium">正文</th>
              <th className="px-3 py-2 font-medium">状态</th>
              <th className="w-16 px-2 py-2" />
            </tr>
          </thead>
          <tbody>
            {matrix.rows.map((row) => (
              <tr key={row.chapter} className="border-b border-studio-border/50 hover:bg-studio-bg/40">
                {selectable && (
                  <td className="px-2 py-2">
                    <input
                      type="checkbox"
                      checked={selectedChapters?.includes(row.chapter) ?? false}
                      disabled={!row.has_body}
                      onChange={() => onToggleChapter?.(row.chapter)}
                    />
                  </td>
                )}
                <td className="whitespace-nowrap px-3 py-2 tabular-nums">{row.chapter}</td>
                <td className="max-w-[180px] px-3 py-2">
                  <div className="truncate font-medium">
                    {row.title || "—"}
                    {!row.in_outline && (
                      <span className="ml-1 font-normal text-[10px] text-studio-muted">(无卷纲)</span>
                    )}
                  </div>
                  {row.outline_preview && (
                    <div className="truncate text-[10px] text-studio-muted">{row.outline_preview}</div>
                  )}
                </td>
                <td className="whitespace-nowrap px-3 py-2 tabular-nums">
                  {row.has_body ? `${row.word_count} 字` : "—"}
                </td>
                <td className={`px-3 py-2 ${matchClass[row.match_status] ?? ""}`}>
                  {matchLabels[row.match_status] ?? row.match_status}
                </td>
                <td className="px-2 py-2">
                  <div className="flex gap-1">
                    {row.has_body && onOpenChapter && (
                      <button
                        type="button"
                        title="打开正文"
                        onClick={() => onOpenChapter(row.chapter)}
                        className="rounded p-1 text-studio-muted hover:bg-studio-panel hover:text-studio-accent"
                      >
                        <ExternalLink className="h-3.5 w-3.5" />
                      </button>
                    )}
                    {row.match_status === "unwritten" && onInsertAfter && (
                      <button
                        type="button"
                        onClick={() => onInsertAfter(row.chapter > 0 ? row.chapter - 1 : 0)}
                        className="rounded px-1.5 py-0.5 text-[10px] text-studio-accent hover:bg-studio-accent/10"
                      >
                        插入
                      </button>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        )}
      </div>
    </div>
  );
}
