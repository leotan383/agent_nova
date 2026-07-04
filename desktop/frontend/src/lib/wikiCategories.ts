import { SettingCategoryDTO, WikiEntryDTO } from "./wails";

/** 设定分类 id（内置或用户自定义） */
export type SettingCategory = string;

export const HIDDEN_SETTING_CATEGORY = "其他";

export const META_SYNOPSIS_ID = "meta:synopsis";

const LEGACY_SUBDIR_TO_CATEGORY: Record<string, string> = {
  角色: "角色",
  世界: "世界观",
  势力: "势力",
  地点: "地点",
  物品: "物品",
  其他: HIDDEN_SETTING_CATEGORY,
};

/** 由 API 分类列表构建 子目录 → 分类 id */
export function buildSubdirToCategory(categories: SettingCategoryDTO[]): Record<string, string> {
  const map: Record<string, string> = { ...LEGACY_SUBDIR_TO_CATEGORY };
  for (const c of categories) {
    map[c.subdir] = c.id;
  }
  return map;
}

/** 由 API 分类列表构建 分类 id → 子目录 */
export function buildCategoryToSubdir(categories: SettingCategoryDTO[]): Record<string, string> {
  const map: Record<string, string> = { [HIDDEN_SETTING_CATEGORY]: "其他" };
  for (const c of categories) {
    map[c.id] = c.subdir;
  }
  return map;
}

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

export function classifySettingEntry(
  e: WikiEntryDTO,
  subdirToCategory: Record<string, string>,
): SettingCategory {
  const rel = settingRelFromID(e.id);
  if (rel) {
    const sub = rel.split("/")[0];
    if (subdirToCategory[sub]) return subdirToCategory[sub];
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
    return "世界观";
  }
  if (title.includes("势力")) return "势力";
  if (["主角", "角色", "人物", "反派", "配角"].some((k) => title.includes(k))) {
    return "角色";
  }
  return HIDDEN_SETTING_CATEGORY;
}

export function countByCategory(
  entries: WikiEntryDTO[],
  categories: SettingCategoryDTO[],
): Record<string, number> {
  const subdirToCategory = buildSubdirToCategory(categories);
  const counts: Record<string, number> = {};
  for (const c of categories) counts[c.id] = 0;
  for (const e of entries) {
    if (e.group === "大纲" || e.kind === "outline") continue;
    const cat = classifySettingEntry(e, subdirToCategory);
    if (cat in counts) counts[cat]++;
  }
  return counts;
}

export function filterByCategory(
  entries: WikiEntryDTO[],
  category: SettingCategory,
  subdirToCategory: Record<string, string>,
): WikiEntryDTO[] {
  return entries.filter((e) => {
    if (e.group === "大纲" || e.kind === "outline") return false;
    return classifySettingEntry(e, subdirToCategory) === category;
  });
}

export function sortOutlineEntries(items: WikiEntryDTO[]) {
  return [...items].sort((a, b) => {
    if (a.title.includes("总纲")) return -1;
    if (b.title.includes("总纲")) return 1;
    return a.title.localeCompare(b.title, "zh-CN");
  });
}

export function listOutlineEntries(entries: WikiEntryDTO[]): WikiEntryDTO[] {
  return sortOutlineEntries(
    entries.filter((e) => e.kind === "outline" || e.group === "大纲"),
  );
}

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

export function resolveSettingsCategoryDir(
  entries: WikiEntryDTO[],
  category: SettingCategory,
  categoryToSubdir: Record<string, string>,
): string {
  const sub = categoryToSubdir[category];
  if (!sub) return "";
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

export function splitCategoryEntries(
  entries: WikiEntryDTO[],
  category: SettingCategory,
  subdirToCategory: Record<string, string>,
) {
  const filtered = filterByCategory(entries, category, subdirToCategory);
  const archives = filtered.filter((e) => e.kind === "setting");
  const states = filtered.filter((e) => e.kind === "entity");
  return { archives, states };
}

export function categoryLabel(
  categoryId: string,
  categories: SettingCategoryDTO[],
): string {
  return categories.find((c) => c.id === categoryId)?.label ?? categoryId;
}

export function categorySubdir(
  categoryId: string,
  categories: SettingCategoryDTO[],
): string {
  return buildCategoryToSubdir(categories)[categoryId] ?? categoryId;
}

export function isBuiltinCategory(
  categoryId: string,
  categories: SettingCategoryDTO[],
): boolean {
  return categories.find((c) => c.id === categoryId)?.builtin ?? false;
}
