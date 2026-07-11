import { useEffect, useRef, useState } from "react";
import { ArrowRight, Lightbulb, MoreHorizontal, PenLine, Pin } from "lucide-react";
import { InspirationCard, formatRelativeTime } from "../lib/wails";
import { coverClassForGenre } from "../lib/libraryUtils";
import { inspirationStatusLabel } from "../lib/inspirationUtils";

type Props = {
  item: InspirationCard;
  onEdit: () => void;
  onCreateNovel: () => void;
  onTogglePin: () => void;
  onToggleArchive: () => void;
  onDelete: () => void;
  onOpenNovel?: () => void;
};

export default function InspirationCardView({
  item,
  onEdit,
  onCreateNovel,
  onTogglePin,
  onToggleArchive,
  onDelete,
  onOpenNovel,
}: Props) {
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const initial = (item.title || "?").charAt(0);
  const cover = coverClassForGenre(item.genre || "其他");
  const status = inspirationStatusLabel[item.status] || item.status;

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
    item.genre || "未分类",
    item.tags?.length ? item.tags.slice(0, 2).join(" · ") : null,
  ]
    .filter(Boolean)
    .join(" · ");

  return (
    <article
      className="group relative flex flex-col overflow-hidden rounded-2xl border border-studio-border/80 bg-studio-panel transition duration-200 hover:border-studio-border hover:shadow-card"
    >
      <div
        className={`relative h-32 cursor-pointer bg-gradient-to-br ${cover}`}
        onClick={onEdit}
        onKeyDown={(e) => e.key === "Enter" && onEdit()}
        role="button"
        tabIndex={0}
      >
        <div className="absolute inset-0 bg-gradient-to-t from-black/45 via-transparent to-transparent" />
        <div className="absolute left-3 top-3 flex flex-wrap gap-1.5">
          <span className="inline-flex items-center gap-1 rounded-full bg-black/35 px-2 py-0.5 text-[10px] text-white/90 backdrop-blur-sm">
            <Lightbulb className="h-3 w-3" />
            {status}
          </span>
          {item.archived && (
            <span className="rounded-full bg-black/40 px-2 py-0.5 text-[10px] text-white/90 backdrop-blur-sm">
              已归档
            </span>
          )}
        </div>
        <div className="absolute bottom-0 left-0 right-0 p-4">
          <div className="flex items-end justify-between gap-2">
            <div className="min-w-0">
              <div className="flex items-center gap-1.5">
                {item.pinned && <Pin className="h-3 w-3 shrink-0 text-white/70" />}
                <h3 className="truncate font-medium text-white">{item.title || "未命名灵感"}</h3>
              </div>
              {subtitle && <p className="mt-0.5 truncate text-xs text-white/65">{subtitle}</p>}
            </div>
            <span className="shrink-0 font-serif text-3xl text-white/20">{initial}</span>
          </div>
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
              <button
                type="button"
                onClick={() => {
                  setMenuOpen(false);
                  onEdit();
                }}
                className="block w-full px-3 py-2 text-left text-sm hover:bg-studio-bg"
              >
                编辑
              </button>
              {item.status !== "used" && (
                <button
                  type="button"
                  onClick={() => {
                    setMenuOpen(false);
                    onCreateNovel();
                  }}
                  className="block w-full px-3 py-2 text-left text-sm hover:bg-studio-bg"
                >
                  创建小说
                </button>
              )}
              {item.novel_id && onOpenNovel && (
                <button
                  type="button"
                  onClick={() => {
                    setMenuOpen(false);
                    onOpenNovel();
                  }}
                  className="block w-full px-3 py-2 text-left text-sm hover:bg-studio-bg"
                >
                  打开关联的书
                </button>
              )}
              <button
                type="button"
                onClick={() => {
                  setMenuOpen(false);
                  onTogglePin();
                }}
                className="block w-full px-3 py-2 text-left text-sm hover:bg-studio-bg"
              >
                {item.pinned ? "取消置顶" : "置顶"}
              </button>
              <button
                type="button"
                onClick={() => {
                  setMenuOpen(false);
                  onToggleArchive();
                }}
                className="block w-full px-3 py-2 text-left text-sm hover:bg-studio-bg"
              >
                {item.archived ? "取消归档" : "归档"}
              </button>
              <div className="my-1 border-t border-studio-border/60" />
              <button
                type="button"
                onClick={() => {
                  setMenuOpen(false);
                  onDelete();
                }}
                className="block w-full px-3 py-2 text-left text-sm text-[rgb(var(--studio-danger-fg))] hover:bg-[rgb(var(--studio-danger-bg))]"
              >
                删除
              </button>
            </div>
          )}
        </div>
      </div>

      <div className="space-y-3 p-4">
        <p className="line-clamp-2 text-xs leading-relaxed text-studio-muted">{item.spark_preview}</p>
        <div className="flex items-center justify-between text-[11px] text-studio-muted">
          {item.novel_title ? (
            <span className="truncate">→ {item.novel_title}</span>
          ) : (
            <span>{status}</span>
          )}
          <span>{formatRelativeTime(item.updated_at)}</span>
        </div>
        {item.status !== "used" && (
          <button
            type="button"
            onClick={onCreateNovel}
            className="flex w-full items-center justify-center gap-1.5 rounded-xl bg-studio-accent/10 py-2 text-xs font-medium text-studio-accent transition hover:bg-studio-accent/15"
          >
            <PenLine className="h-3.5 w-3.5" />
            用这个灵感创建
            <ArrowRight className="h-3.5 w-3.5" />
          </button>
        )}
      </div>
    </article>
  );
}
