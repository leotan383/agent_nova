import { WikiEntryDTO } from "./wails";

export const SETTING_CATEGORIES = ["角色", "背景", "势力", "地点", "物品", "其他"] as const;
export type SettingCategory = (typeof SETTING_CATEGORIES)[number];

export const META_SYNOPSIS_ID = "meta:synopsis";

/** UI 分类 → 设定集子目录 */
export const CATEGORY_TO_SUBDIR: Record<SettingCategory, string> = {
  角色: "角色",
  背景: "世界",
  势力: "势力",
  地点: "地点",
  物品: "物品",
  其他: "其他",
};

const SUBDIR_TO_CATEGORY: Record<string, SettingCategory> = {
  角色: "角色",
  世界: "背景",
  势力: "势力",
  地点: "地点",
  物品: "物品",
  其他: "其他",
};

export function settingRelFromID(id: string): string | null {
  if (!id.startsWith("setting:")) return null;
  return id.slice("setting:".length);
}

export function parseEntityTypeFromWikiID(id: string): string | null {
  if (!id.startsWith("entity:")) return null;
  const rest = id.slice("entity:".length);
  const colon = rest.indexOf(":");
  return colon >= 0 ? rest.slice(0, colon) : rest;
}

export function classifySettingEntry(e: WikiEntryDTO): SettingCategory {
  const rel = settingRelFromID(e.id);
  if (rel) {
    const sub = rel.split("/")[0];
    if (SUBDIR_TO_CATEGORY[sub]) return SUBDIR_TO_CATEGORY[sub];
  }
  if (e.group === "人物") return "角色";
  if (e.kind === "entity") {
    const typ = parseEntityTypeFromWikiID(e.id);
    if (typ === "character") return "角色";
    if (typ === "location") return "地点";
    if (typ === "item") return "物品";
  }
  const title = e.title;
  if (["世界观", "背景", "力量", "科技", "体系", "设定"].some((k) => title.includes(k))) {
    return "背景";
  }
  if (title.includes("势力")) return "势力";
  if (["主角", "角色", "人物", "反派", "配角"].some((k) => title.includes(k))) {
    return "角色";
  }
  return "其他";
}

export function countByCategory(entries: WikiEntryDTO[]): Record<SettingCategory, number> {
  const counts: Record<SettingCategory, number> = {
    角色: 0,
    背景: 0,
    势力: 0,
    地点: 0,
    物品: 0,
    其他: 0,
  };
  for (const e of entries) {
    if (e.group === "大纲" || e.kind === "outline") continue;
    counts[classifySettingEntry(e)]++;
  }
  return counts;
}

export function filterByCategory(entries: WikiEntryDTO[], category: SettingCategory): WikiEntryDTO[] {
  return entries.filter((e) => {
    if (e.group === "大纲" || e.kind === "outline") return false;
    return classifySettingEntry(e) === category;
  });
}

export function sortOutlineEntries(items: WikiEntryDTO[]) {
  return [...items].sort((a, b) => {
    if (a.title.includes("总纲")) return -1;
    if (b.title.includes("总纲")) return 1;
    return a.title.localeCompare(b.title, "zh-CN");
  });
}

/** 大纲目录下的 Markdown 条目（不含简介 meta） */
export function listOutlineEntries(entries: WikiEntryDTO[]): WikiEntryDTO[] {
  return sortOutlineEntries(
    entries.filter((e) => e.kind === "outline" || e.group === "大纲"),
  );
}

/** 侧边栏「大纲」下展示的条目：隐藏总纲、卷纲、爽点规划（卷纲由卷纲规划页管理） */
export function isVolumeOutlineTitle(title: string): boolean {
  return /^第\s*\d+\s*卷/u.test(title.trim());
}

export function listSidebarOutlineEntries(entries: WikiEntryDTO[]): WikiEntryDTO[] {
  return listOutlineEntries(entries).filter((e) => {
    if (e.title.includes("总纲")) return false;
    if (e.title.includes("爽点规划")) return false;
    if (isVolumeOutlineTitle(e.title)) return false;
    return true;
  });
}

/** 解析设定集某分类的磁盘目录（用于新建文件） */
export function resolveSettingsCategoryDir(
  entries: WikiEntryDTO[],
  category: SettingCategory,
): string {
  const sub = CATEGORY_TO_SUBDIR[category];
  const hit = entries.find((e) => {
    if (e.kind !== "setting" || !e.path) return false;
    return e.path.replace(/\\/g, "/").includes(`/设定集/${sub}/`);
  });
  if (hit?.path) {
    const norm = hit.path.replace(/\\/g, "/");
    const needle = `/设定集/${sub}`;
    const idx = norm.indexOf(needle);
    if (idx >= 0) {
      const sep = hit.path.includes("\\") ? "\\" : "/";
      const base = hit.path.slice(0, idx + needle.length);
      return sep === "\\" ? base.replace(/\//g, "\\") : base;
    }
  }
  const any = entries.find((e) => e.kind === "setting" && e.path);
  if (!any?.path) return "";
  const sep = any.path.includes("\\") ? "\\" : "/";
  const norm = any.path.replace(/\\/g, "/");
  const marker = "/设定集";
  const idx = norm.indexOf(marker);
  if (idx < 0) return "";
  const root = any.path.slice(0, idx + marker.length);
  return `${root}${sep}${sub}`;
}

export function splitCategoryEntries(entries: WikiEntryDTO[], category: SettingCategory) {
  const filtered = filterByCategory(entries, category);
  const archives = filtered.filter((e) => e.kind === "setting");
  const states = filtered.filter((e) => e.kind === "entity");
  return { archives, states };
}
