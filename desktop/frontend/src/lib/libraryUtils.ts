import { NovelCard } from "./wails";

export type LibrarySortKey = "last_opened" | "progress" | "chapters" | "title";

export type LibraryFilters = {
  query: string;
  genre: string;
  phase: string;
  showArchived: boolean;
};

export const genreCoverClass: Record<string, string> = {
  玄幻: "from-violet-950 via-purple-900/90 to-indigo-950",
  仙侠: "from-teal-950 via-emerald-900/80 to-cyan-950",
  都市: "from-slate-700 via-slate-600/90 to-zinc-800",
  科幻: "from-blue-950 via-indigo-900/80 to-slate-950",
  历史: "from-amber-950 via-stone-800/90 to-neutral-900",
  游戏: "from-fuchsia-950 via-purple-900/80 to-violet-950",
  悬疑: "from-neutral-900 via-zinc-800/90 to-stone-950",
  其他: "from-studio-cover-from via-studio-cover-via to-studio-cover-to",
};

export function coverClassForGenre(genre: string): string {
  return genreCoverClass[genre] ?? genreCoverClass["其他"];
}

export function nextWriteChapter(novel: NovelCard): number {
  return Math.max(1, novel.current_chapter + 1);
}

export function filterNovels(novels: NovelCard[], filters: LibraryFilters): NovelCard[] {
  const q = filters.query.trim().toLowerCase();
  return novels.filter((n) => {
    if (!filters.showArchived && n.archived) return false;
    if (filters.showArchived && !n.archived) return false;
    if (q && !n.title.toLowerCase().includes(q) && !n.genre.toLowerCase().includes(q)) {
      return false;
    }
    if (filters.genre && n.genre !== filters.genre) return false;
    if (filters.phase && n.phase !== filters.phase) return false;
    return true;
  });
}

export function sortNovels(novels: NovelCard[], sort: LibrarySortKey): NovelCard[] {
  const list = [...novels];
  list.sort((a, b) => {
    if (a.pinned !== b.pinned) return a.pinned ? -1 : 1;
    switch (sort) {
      case "progress":
        return (b.progress_percent ?? 0) - (a.progress_percent ?? 0);
      case "chapters":
        return b.chapter_count - a.chapter_count;
      case "title":
        return (a.title || "").localeCompare(b.title || "", "zh-CN");
      case "last_opened":
      default:
        return new Date(b.last_opened_at || 0).getTime() - new Date(a.last_opened_at || 0).getTime();
    }
  });
  return list;
}

export function splitLibrarySections(novels: NovelCard[], showArchived: boolean) {
  if (showArchived) {
    return { pinned: [] as NovelCard[], main: novels, archived: [] as NovelCard[] };
  }
  const pinned = novels.filter((n) => n.pinned);
  const main = novels.filter((n) => !n.pinned);
  return { pinned, main, archived: [] as NovelCard[] };
}

export function libraryStats(novels: NovelCard[]) {
  const active = novels.filter((n) => !n.archived && !n.missing);
  return {
    count: active.length,
    totalWords: active.reduce((sum, n) => sum + (n.written_words ?? 0), 0),
    inProgress: active.filter((n) => n.phase === "writing" || n.phase === "planning").length,
  };
}
