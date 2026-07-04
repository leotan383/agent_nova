import { SettingCategoryDTO, SettingChecklistDTO, app } from "./wails";

export type { SettingChecklistDTO };

export async function fetchSettingChecklist(): Promise<SettingChecklistDTO> {
  return app().GetSettingChecklist();
}

export function checklistItemsForCategory(
  checklist: SettingChecklistDTO | null,
  categoryId: string,
) {
  if (!checklist) return [];
  return checklist.items.filter((it) => it.category_id === categoryId);
}

export function missingChecklistItems(checklist: SettingChecklistDTO | null) {
  if (!checklist) return [];
  return checklist.items.filter((it) => !it.done);
}

export async function saveCategoryOrder(categories: SettingCategoryDTO[]) {
  const order = categories.map((c) => c.id);
  await app().SaveSettingCategoryOrder(order);
}

export function reorderCategories(
  categories: SettingCategoryDTO[],
  fromId: string,
  toId: string,
): SettingCategoryDTO[] {
  const fromIdx = categories.findIndex((c) => c.id === fromId);
  const toIdx = categories.findIndex((c) => c.id === toId);
  if (fromIdx < 0 || toIdx < 0 || fromIdx === toIdx) return categories;
  const next = [...categories];
  const [moved] = next.splice(fromIdx, 1);
  next.splice(toIdx, 0, moved);
  return next;
}

export function pinCategory(
  categories: SettingCategoryDTO[],
  categoryId: string,
): SettingCategoryDTO[] {
  const idx = categories.findIndex((c) => c.id === categoryId);
  if (idx <= 0) return categories;
  const next = [...categories];
  const [moved] = next.splice(idx, 1);
  next.unshift(moved);
  return next;
}
