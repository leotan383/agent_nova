import { useEffect, useRef, useState } from "react";
import { ArrowRight, MoreHorizontal, PenLine, Pin } from "lucide-react";
import { NovelCard, formatRelativeTime, formatWordCount, phaseLabel } from "../lib/wails";
import { coverClassForGenre, nextWriteChapter } from "../lib/libraryUtils";

type Props = {
  novel: NovelCard;
  continueActive?: boolean;
  onOpen: () => void;
  onContinueWrite?: () => void;
  onReveal: () => void;
  onRemove: () => void;
  onTogglePin: () => void;
  onToggleArchive: () => void;
};

export default function NovelCardView({
  novel,
  continueActive = false,
  onOpen,
  onContinueWrite,
  onReveal,
  onRemove,
  onTogglePin,
  onToggleArchive,
}: Props) {
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const initial = (novel.title || "?").charAt(0);
  const phase = phaseLabel[novel.phase] || novel.phase || "未知";
  const cover = coverClassForGenre(novel.genre || "其他");
  const progress = Math.min(100, novel.progress_percent ?? 0);
  const nextCh = nextWriteChapter(novel);

  useEffect(() => {
    if (!menuOpen) return;
    const close = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false);
      }
    };
    document.addEventListener("mousedown", close);
    return () => document.removeEventListener("mousedown", close);
  }, [menuOpen]);

  const subtitle = [
    novel.genre || "未分类",
    novel.chapter_count > 0 ? `${novel.chapter_count} 章` : null,
    novel.current_chapter > 0 ? `写到第 ${novel.current_chapter} 章` : null,
  ]
    .filter(Boolean)
    .join(" · ");

  return (
    <article
      className={`group relative flex flex-col overflow-hidden rounded-2xl border bg-studio-panel transition duration-200 hover:shadow-card ${
        continueActive
          ? "border-studio-accent/35 ring-1 ring-studio-accent/15 hover:border-studio-accent/45"
          : "border-studio-border/80 hover:border-studio-border"
      } ${novel.missing ? "opacity-50" : "cursor-pointer"}`}
      onClick={() => {
        if (!novel.missing) onOpen();
      }}
      onKeyDown={(e) => {
        if (e.key === "Enter" && !novel.missing) onOpen();
      }}
      role="button"
      tabIndex={0}
    >
      <div className={`relative bg-gradient-to-br ${cover} ${continueActive ? "h-36" : "h-32"}`}>
        <div className="absolute inset-0 bg-gradient-to-t from-black/40 via-transparent to-transparent" />
        <div className="absolute bottom-0 left-0 right-0 p-4">
          <div className="flex items-end justify-between gap-2">
            <div className="min-w-0">
              <div className="flex items-center gap-1.5">
                {novel.pinned && <Pin className="h-3 w-3 shrink-0 text-white/70" />}
                <h3 className="truncate font-medium text-white">{novel.title || "未命名"}</h3>
              </div>
              <p className="mt-0.5 truncate text-xs text-white/65">{subtitle}</p>
            </div>
            <span className="shrink-0 font-serif text-3xl text-white/20">{initial}</span>
          </div>
        </div>

        <div className="absolute left-3 top-3 flex flex-wrap gap-1.5">
          {continueActive && !novel.missing && (
            <span className="rounded-full bg-studio-accent/90 px-2 py-0.5 text-[10px] font-medium text-studio-on-accent">
              继续创作
            </span>
          )}
          {(novel.missing || novel.archived) && (
            <span className="rounded-full bg-black/40 px-2 py-0.5 text-[10px] text-white/90 backdrop-blur-sm">
              {novel.missing ? "路径失效" : "已归档"}
            </span>
          )}
        </div>

        <div
          ref={menuRef}
          className="absolute right-2 top-2 opacity-0 transition group-hover:opacity-100"
          onClick={(e) => e.stopPropagation()}
        >
          <button
            type="button"
            onClick={() => setMenuOpen((v) => !v)}
            className="rounded-lg bg-black/30 p-1.5 text-white/90 backdrop-blur-sm hover:bg-black/45"
          >
            <MoreHorizontal className="h-4 w-4" />
          </button>
          {menuOpen && (
            <div className="absolute right-0 top-full z-20 mt-1 w-40 overflow-hidden rounded-xl border border-studio-border bg-studio-panel py-1 text-studio-text shadow-card">
              {onContinueWrite && !novel.missing && !novel.archived && (
                <button
                  type="button"
                  onClick={() => {
                    setMenuOpen(false);
                    onContinueWrite();
                  }}
                  className="block w-full px-3 py-2 text-left text-sm hover:bg-studio-bg"
                >
                  继续写章
                </button>
              )}
              <button
                type="button"
                onClick={() => {
                  setMenuOpen(false);
                  onReveal();
                }}
                className="block w-full px-3 py-2 text-left text-sm hover:bg-studio-bg"
              >
                打开文件夹
              </button>
              <button
                type="button"
                onClick={() => {
                  setMenuOpen(false);
                  onTogglePin();
                }}
                className="block w-full px-3 py-2 text-left text-sm hover:bg-studio-bg"
              >
                {novel.pinned ? "取消置顶" : "置顶"}
              </button>
              <button
                type="button"
                onClick={() => {
                  setMenuOpen(false);
                  onToggleArchive();
                }}
                className="block w-full px-3 py-2 text-left text-sm hover:bg-studio-bg"
              >
                {novel.archived ? "取消归档" : "归档"}
              </button>
              <div className="my-1 border-t border-studio-border/60" />
              <button
                type="button"
                onClick={() => {
                  setMenuOpen(false);
                  onRemove();
                }}
                className="block w-full px-3 py-2 text-left text-sm text-[rgb(var(--studio-danger-fg))] hover:bg-[rgb(var(--studio-danger-bg))]"
              >
                从书库移除
              </button>
            </div>
          )}
        </div>
      </div>

      <div className="space-y-3 p-4">
        {novel.target_words != null && novel.target_words > 0 ? (
          <div>
            <div className="mb-1.5 flex items-center justify-between text-[11px] text-studio-muted">
              <span>{formatWordCount(novel.written_words ?? 0)}</span>
              <span>{progress.toFixed(0)}%</span>
            </div>
            <div className="h-1 overflow-hidden rounded-full bg-studio-bg">
              <div className="h-full rounded-full bg-studio-accent/70" style={{ width: `${progress}%` }} />
            </div>
          </div>
        ) : (
          <p className="text-[11px] text-studio-muted">{phase}</p>
        )}

        <div className="flex items-center justify-between text-[11px] text-studio-muted">
          <span>{phase}</span>
          <span>{formatRelativeTime(novel.last_opened_at)}</span>
        </div>

        {continueActive && onContinueWrite && !novel.missing && !novel.archived && (
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              onContinueWrite();
            }}
            className="flex w-full items-center justify-center gap-1.5 rounded-xl bg-studio-accent/10 py-2 text-xs font-medium text-studio-accent transition hover:bg-studio-accent/15"
          >
            <PenLine className="h-3.5 w-3.5" />
            写第 {nextCh} 章
            <ArrowRight className="h-3.5 w-3.5" />
          </button>
        )}
      </div>
    </article>
  );
}
