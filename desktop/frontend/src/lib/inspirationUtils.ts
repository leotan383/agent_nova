import { InspirationCard, InspirationPrefillDTO } from "./wails";

export type InspirationSortKey = "updated" | "created" | "title";

export type InspirationFilters = {
  query: string;
  genre: string;
  status: string;
  showArchived: boolean;
};

export const inspirationStatusLabel: Record<string, string> = {
  seed: "种子",
  developing: "完善中",
  ready: "可开书",
  used: "已开书",
  archived: "已归档",
};

export const inspirationStatusOptions = [
  { value: "", label: "全部状态" },
  { value: "seed", label: "种子" },
  { value: "developing", label: "完善中" },
  { value: "ready", label: "可开书" },
  { value: "used", label: "已开书" },
];

export function filterInspirations(items: InspirationCard[], filters: InspirationFilters): InspirationCard[] {
  const q = filters.query.trim().toLowerCase();
  return items.filter((item) => {
    if (!filters.showArchived && item.archived) return false;
    if (filters.showArchived && !item.archived) return false;
    if (filters.status && item.status !== filters.status) return false;
    if (filters.genre && item.genre !== filters.genre) return false;
    if (
      q &&
      !item.title.toLowerCase().includes(q) &&
      !item.spark_preview.toLowerCase().includes(q) &&
      !(item.genre || "").toLowerCase().includes(q) &&
      !(item.tags || []).some((t) => t.toLowerCase().includes(q))
    ) {
      return false;
    }
    return true;
  });
}

export function sortInspirations(items: InspirationCard[], sort: InspirationSortKey): InspirationCard[] {
  const list = [...items];
  list.sort((a, b) => {
    if (a.pinned !== b.pinned) return a.pinned ? -1 : 1;
    switch (sort) {
      case "title":
        return (a.title || "").localeCompare(b.title || "", "zh-CN");
      case "created":
        return new Date(b.created_at || 0).getTime() - new Date(a.created_at || 0).getTime();
      case "updated":
      default:
        return new Date(b.updated_at || 0).getTime() - new Date(a.updated_at || 0).getTime();
    }
  });
  return list;
}

export function inspirationStats(items: InspirationCard[]) {
  const active = items.filter((i) => !i.archived);
  return {
    count: active.length,
    ready: active.filter((i) => i.status === "ready").length,
    used: active.filter((i) => i.status === "used").length,
  };
}

export function prefillToCreateForm(prefill: InspirationPrefillDTO) {
  return {
    title: prefill.title,
    genre: prefill.genre || "玄幻",
    style: prefill.style || "热血",
    synopsis: prefill.synopsis,
    protagonist: prefill.protagonist,
    cheat: prefill.cheat,
  };
}

export function parseTagsInput(raw: string): string[] {
  return raw
    .split(/[,，、\s]+/)
    .map((t) => t.trim())
    .filter(Boolean);
}

export function tagsToInput(tags: string[] | undefined): string {
  return (tags || []).join("、");
}
