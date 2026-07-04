import { useEffect, useState } from "react";
import { Check, Download, ExternalLink, Loader2, X } from "lucide-react";
import { ExportResultDTO, StatusReport, app } from "../lib/wails";

type Props = {
  open: boolean;
  onClose: () => void;
  status: StatusReport | null;
};

const formats = [
  { id: "markdown", label: "Markdown 合集", hint: "适合备份与二次编辑" },
  { id: "epub", label: "EPUB 电子书", hint: "可在 Apple Books / Kindle 阅读" },
  { id: "pdf", label: "PDF 文档", hint: "适合打印与分享，需系统有中文字体" },
  { id: "txt", label: "纯文本 TXT", hint: "干净正文，便于投稿平台" },
] as const;

export default function ExportDialog({ open, onClose, status }: Props) {
  const [format, setFormat] = useState<(typeof formats)[number]["id"]>("markdown");
  const [outPath, setOutPath] = useState("");
  const [fromChapter, setFromChapter] = useState(0);
  const [toChapter, setToChapter] = useState(0);
  const [exporting, setExporting] = useState(false);
  const [error, setError] = useState("");
  const [result, setResult] = useState<ExportResultDTO | null>(null);

  useEffect(() => {
    if (!open) return;
    setError("");
    setResult(null);
    setOutPath("");
    setFromChapter(0);
    setToChapter(status?.chapter_count ?? 0);
  }, [open, status?.chapter_count]);

  const pickPath = async () => {
    setError("");
    try {
      const defaultName = await app().DefaultExportFilename(format);
      const path = await app().PickExportPath(format, defaultName);
      if (path) setOutPath(path);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const doExport = async () => {
    if (!outPath.trim()) {
      setError("请先选择保存位置");
      return;
    }
    setExporting(true);
    setError("");
    try {
      const res = await app().ExportProject({
        format,
        out_path: outPath.trim(),
        from_chapter: fromChapter,
        to_chapter: toChapter,
      });
      setResult(res);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setExporting(false);
    }
  };

  if (!open) return null;

  return (
    <div className="studio-modal-overlay fixed inset-0 z-50 flex items-center justify-center p-4" onClick={onClose}>
      <div
        className="w-full max-w-lg rounded-xl border border-studio-border bg-studio-panel shadow-card"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between border-b border-studio-border px-5 py-4">
          <div className="flex items-center gap-2">
            <Download className="h-4 w-4 text-studio-accent" />
            <h2 className="text-base font-medium">导出小说</h2>
          </div>
          <button type="button" onClick={onClose} className="rounded p-1 text-studio-muted hover:text-studio-text">
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="space-y-4 p-5">
          {result ? (
            <div className="rounded-lg border border-[rgb(var(--studio-diff-add-stat)/0.3)] bg-[rgb(var(--studio-diff-add-bg))] p-4">
              <div className="flex items-center gap-2 text-sm font-medium text-[rgb(var(--studio-diff-add-stat))]">
                <Check className="h-4 w-4" />
                导出成功
              </div>
              <p className="mt-2 break-all text-xs text-studio-muted">{result.path}</p>
              <p className="mt-2 text-xs text-studio-muted">
                {result.chapter_count} 章 · 约 {result.word_count.toLocaleString()} 字
              </p>
              <button
                type="button"
                onClick={() => app().RevealInFolder(result.path)}
                className="mt-3 inline-flex items-center gap-1 text-xs text-studio-accent hover:underline"
              >
                <ExternalLink className="h-3 w-3" />
                在文件夹中显示
              </button>
            </div>
          ) : (
            <>
              <div>
                <label className="mb-2 block text-xs text-studio-muted">导出格式</label>
                <div className="space-y-2">
                  {formats.map((f) => (
                    <label
                      key={f.id}
                      className={`flex cursor-pointer items-start gap-3 rounded-lg border px-3 py-2.5 transition ${
                        format === f.id
                          ? "border-studio-accent/50 bg-studio-accent/5"
                          : "border-studio-border hover:border-studio-muted"
                      }`}
                    >
                      <input
                        type="radio"
                        name="format"
                        checked={format === f.id}
                        onChange={() => setFormat(f.id)}
                        className="mt-1"
                      />
                      <div>
                        <p className="text-sm font-medium">{f.label}</p>
                        <p className="text-xs text-studio-muted">{f.hint}</p>
                      </div>
                    </label>
                  ))}
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="mb-1 block text-xs text-studio-muted">起始章（0=不限）</label>
                  <input
                    type="number"
                    min={0}
                    value={fromChapter || ""}
                    onChange={(e) => setFromChapter(Number(e.target.value) || 0)}
                    className="w-full rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none"
                  />
                </div>
                <div>
                  <label className="mb-1 block text-xs text-studio-muted">结束章（0=不限）</label>
                  <input
                    type="number"
                    min={0}
                    value={toChapter || ""}
                    onChange={(e) => setToChapter(Number(e.target.value) || 0)}
                    placeholder={status?.chapter_count ? String(status.chapter_count) : ""}
                    className="w-full rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none"
                  />
                </div>
              </div>

              <div>
                <label className="mb-1 block text-xs text-studio-muted">保存位置</label>
                <div className="flex gap-2">
                  <input
                    value={outPath}
                    readOnly
                    placeholder="点击右侧选择保存路径…"
                    className="min-w-0 flex-1 rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none"
                  />
                  <button
                    type="button"
                    onClick={pickPath}
                    className="shrink-0 rounded-lg border border-studio-border px-3 text-sm hover:bg-studio-bg"
                  >
                    选择…
                  </button>
                </div>
              </div>

              {status && status.chapter_count === 0 && (
                <p className="text-xs text-[rgb(var(--studio-warning-fg))]">当前尚无章节正文，无法导出。</p>
              )}
            </>
          )}

          {error && <div className="studio-alert-error-compact">{error}</div>}
        </div>

        <div className="flex justify-end gap-2 border-t border-studio-border px-5 py-4">
          <button type="button" onClick={onClose} className="rounded-lg px-4 py-2 text-sm text-studio-muted">
            {result ? "关闭" : "取消"}
          </button>
          {!result && (
            <button
              type="button"
              onClick={doExport}
              disabled={exporting || !outPath.trim() || (status?.chapter_count ?? 0) === 0}
              className="inline-flex items-center gap-1 rounded-lg bg-studio-accent px-4 py-2 text-sm text-studio-on-accent disabled:opacity-40"
            >
              {exporting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Download className="h-4 w-4" />}
              导出
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
