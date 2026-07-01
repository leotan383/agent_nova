import { FileText, History, PenLine, Wand2 } from "lucide-react";
import { ChapterDTO, StatusReport } from "../lib/wails";
import ChapterCoachPanel from "./ChapterCoachPanel";
import ChapterDocumentPanel from "./ChapterDocumentPanel";
import ChapterStatusBadge from "./ChapterStatusBadge";
import ChapterVersionPanel from "./ChapterVersionPanel";
import WritePanel from "./WritePanel";

export type ChapterDocTab = "body" | "review" | "summary";
export type ChaptersView = "read" | "write";

type Props = {
  status: StatusReport | null;
  chapters: ChapterDTO[];
  view: ChaptersView;
  selectedChapter: number | null;
  chapterDocTab: ChapterDocTab;
  chapterRefreshKey: number;
  autoReviewChapter: number | null;
  versionPanelOpen: boolean;
  onVersionPanelOpenChange: (open: boolean) => void;
  onSelectChapter: (num: number) => void;
  onStartWrite: () => void;
  onWriteComplete: () => void;
  onGoToPlanning: (volume?: number) => void;
  onReviewChapter: (num: number) => void;
  onReadChapter: (num: number) => void;
  onChapterSaved: () => void;
  onReviewComplete: () => void;
  onRebuildIndex: () => Promise<void>;
};

export default function ChaptersPanel({
  status,
  chapters,
  view,
  selectedChapter,
  chapterDocTab,
  chapterRefreshKey,
  autoReviewChapter,
  versionPanelOpen,
  onVersionPanelOpenChange,
  onSelectChapter,
  onStartWrite,
  onWriteComplete,
  onGoToPlanning,
  onReviewChapter,
  onReadChapter,
  onChapterSaved,
  onReviewComplete,
  onRebuildIndex,
}: Props) {
  const nextWriteChapter = Math.max(1, (status?.current_chapter ?? 0) + 1);

  return (
    <div className="flex min-h-0 flex-1 gap-4 overflow-hidden">
      <div className="flex w-52 shrink-0 flex-col overflow-hidden rounded-xl border border-studio-border bg-studio-panel">
        <div className="shrink-0 border-b border-studio-border p-3">
          <button
            type="button"
            onClick={onStartWrite}
            className={`flex w-full items-center gap-2 rounded-lg px-3 py-2.5 text-left text-sm transition ${
              view === "write"
                ? "bg-studio-accent text-studio-on-accent shadow-sm"
                : "border border-studio-accent/30 bg-studio-accent/10 text-studio-accent hover:bg-studio-accent/15"
            }`}
          >
            <Wand2 className="h-4 w-4 shrink-0" />
            <span className="min-w-0 flex-1">
              <span className="block font-medium">AI 写章</span>
              <span className={`block text-[10px] ${view === "write" ? "text-studio-on-accent/80" : "text-studio-accent/80"}`}>
                第 {nextWriteChapter} 章
              </span>
            </span>
          </button>
        </div>

        <div className="flex shrink-0 items-center gap-1.5 border-b border-studio-border px-3 py-2">
          <FileText className="h-3.5 w-3.5 text-studio-muted" />
          <span className="text-xs font-medium text-studio-muted">已有章节</span>
          {chapters.length > 0 && (
            <span className="ml-auto text-[10px] tabular-nums text-studio-muted/70">{chapters.length}</span>
          )}
        </div>

        <ul className="min-h-0 flex-1 overflow-y-auto">
          {chapters.length === 0 ? (
            <li className="p-4 text-xs leading-relaxed text-studio-muted">
              还没有章节，点击上方「AI 写章」开始创作。
            </li>
          ) : (
            chapters.map((c) => {
              const active = view === "read" && selectedChapter === c.number;
              return (
                <li key={c.number}>
                  <button
                    type="button"
                    onClick={() => onSelectChapter(c.number)}
                    className={`w-full border-b border-studio-border px-4 py-3 text-left text-sm transition hover:bg-studio-bg ${
                      active ? "bg-studio-bg text-studio-accent" : ""
                    }`}
                  >
                    <div className="flex items-center gap-2">
                      <span className="font-medium">第{c.number}章</span>
                      <ChapterStatusBadge status={c.status} compact />
                    </div>
                    <div className="truncate text-xs text-studio-muted">
                      {c.title || "无标题"} · {c.word_count}字
                    </div>
                  </button>
                </li>
              );
            })
          )}
        </ul>
      </div>

      {view === "write" ? (
        <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden rounded-xl border border-studio-border bg-studio-panel">
          <WritePanel
            embedded
            status={status}
            onComplete={onWriteComplete}
            onGoToPlanning={onGoToPlanning}
            onReviewChapter={onReviewChapter}
            onReadChapter={onReadChapter}
            onRebuildIndex={onRebuildIndex}
          />
        </div>
      ) : (
        <>
          <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden rounded-xl border border-studio-border bg-studio-paper text-studio-ink">
            {selectedChapter && (
              <div className="flex shrink-0 items-center justify-between border-b border-studio-border px-4 py-2">
                <span className="text-sm font-medium">第{selectedChapter}章</span>
                <button
                  type="button"
                  onClick={() => onVersionPanelOpenChange(true)}
                  className="inline-flex items-center gap-1 rounded-lg px-2 py-1 text-xs text-studio-muted hover:bg-studio-bg hover:text-studio-text"
                >
                  <History className="h-3.5 w-3.5" />
                  版本历史
                </button>
              </div>
            )}
            <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
              {selectedChapter ? (
                <ChapterDocumentPanel
                  key={`${selectedChapter}-${chapterRefreshKey}`}
                  chapter={selectedChapter}
                  initialTab={chapterDocTab}
                  autoStartReview={autoReviewChapter === selectedChapter}
                  onSaved={onChapterSaved}
                  onReviewComplete={onReviewComplete}
                />
              ) : (
                <div className="flex h-full flex-col items-center justify-center px-6 text-center">
                  <PenLine className="mb-3 h-8 w-8 text-studio-muted/30" />
                  <p className="text-sm font-medium text-studio-muted">选择章节阅读或改稿</p>
                  <p className="mt-2 max-w-xs text-xs leading-relaxed text-studio-muted/80">
                    左侧列表点已有章节查看正文；点「AI 写章」开始生成新章。
                  </p>
                </div>
              )}
            </div>
          </div>
          {selectedChapter && (
            <ChapterCoachPanel chapter={selectedChapter} onApplied={onChapterSaved} />
          )}
          {selectedChapter && (
            <ChapterVersionPanel
              chapter={selectedChapter}
              open={versionPanelOpen}
              onClose={() => onVersionPanelOpenChange(false)}
              onRestored={onChapterSaved}
            />
          )}
        </>
      )}
    </div>
  );
}
