import { useState, type DragEvent } from "react";
import {
  GripVertical,
  Pencil,
  Pin,
  Plus,
  Trash2,
} from "lucide-react";
import { SettingCategoryDTO } from "../lib/wails";
import {
  pinCategory,
  reorderCategories,
  saveCategoryOrder,
} from "../lib/settingCategories";
import { SettingCategory } from "../lib/wikiCategories";
import { settingCategoryIcon } from "../lib/settingCategoryIcons";

type Props = {
  categories: SettingCategoryDTO[];
  wikiCounts: Record<string, number>;
  activeCategory: SettingCategory | null;
  wikiTabActive: boolean;
  wikiThemeOnly: boolean;
  onSelectCategory: (id: SettingCategory) => void;
  onCreateSetting: (id: SettingCategory) => void;
  onRenameCategory: (id: string) => void;
  onDeleteCategory: (cat: SettingCategoryDTO) => void;
  onOrderChange: (categories: SettingCategoryDTO[]) => void;
};

function CategoryRow({
  cat,
  count,
  active,
  onSelect,
  onCreateSetting,
  onRename,
  onDelete,
  onPin,
  onDragStart,
  onDragOver,
  onDrop,
}: {
  cat: SettingCategoryDTO;
  count: number;
  active: boolean;
  onSelect: () => void;
  onCreateSetting: () => void;
  onRename: () => void;
  onDelete: () => void;
  onPin: () => void;
  onDragStart: () => void;
  onDragOver: (e: DragEvent) => void;
  onDrop: (e: DragEvent) => void;
}) {
  const Icon = settingCategoryIcon(cat.id);
  const displayLabel = cat.label || cat.id;
  return (
    <div
      className="group relative flex items-center rounded-lg"
      draggable
      onDragStart={onDragStart}
      onDragOver={onDragOver}
      onDrop={onDrop}
    >
      <span
        className="absolute -left-0.5 top-1/2 z-10 hidden -translate-y-1/2 cursor-grab text-studio-muted/50 group-hover:block active:cursor-grabbing"
        title="拖拽排序"
      >
        <GripVertical className="h-3.5 w-3.5" />
      </span>
      <button
        type="button"
        onClick={onSelect}
        title={displayLabel}
        className={`flex w-full min-w-0 items-center gap-2 rounded-lg px-2 py-2 text-sm transition group-hover:pl-5 ${
          active
            ? "bg-studio-accent/15 text-studio-accent"
            : "text-studio-muted hover:bg-studio-panel hover:text-studio-text"
        }`}
      >
        <Icon className="h-4 w-4 shrink-0" />
        <span className="min-w-0 flex-1 truncate text-left">{displayLabel}</span>
        {count > 0 && (
          <span className="shrink-0 text-xs tabular-nums text-studio-muted/70">({count})</span>
        )}
      </button>
      <div className="pointer-events-none absolute inset-y-0 right-0 hidden items-center gap-0.5 rounded-r-lg bg-gradient-to-l from-studio-panel via-studio-panel/95 to-transparent pl-6 pr-0.5 group-hover:pointer-events-auto group-hover:flex">
        <button
          type="button"
          onClick={onCreateSetting}
          className="shrink-0 rounded-md p-1 text-studio-muted transition hover:bg-studio-bg hover:text-studio-accent"
          title={`新建${displayLabel}设定`}
        >
          <Plus className="h-3.5 w-3.5" />
        </button>
        <button
          type="button"
          onClick={onPin}
          className="shrink-0 rounded-md p-1 text-studio-muted transition hover:bg-studio-bg hover:text-studio-text"
          title="置顶"
        >
          <Pin className="h-3 w-3" />
        </button>
        {!cat.builtin && (
          <>
            <button
              type="button"
              onClick={onRename}
              className="shrink-0 rounded-md p-1 text-studio-muted transition hover:bg-studio-bg hover:text-studio-text"
              title={`重命名「${displayLabel}」`}
            >
              <Pencil className="h-3 w-3" />
            </button>
            <button
              type="button"
              onClick={onDelete}
              className="shrink-0 rounded-md p-1 text-studio-muted transition hover:bg-[rgb(var(--studio-danger-bg))] hover:text-[rgb(var(--studio-danger-fg))]"
              title={`删除「${displayLabel}」`}
            >
              <Trash2 className="h-3 w-3" />
            </button>
          </>
        )}
      </div>
    </div>
  );
}

export default function SettingCategoryNav({
  categories,
  wikiCounts,
  activeCategory,
  wikiTabActive,
  wikiThemeOnly,
  onSelectCategory,
  onCreateSetting,
  onRenameCategory,
  onDeleteCategory,
  onOrderChange,
}: Props) {
  const [draggingId, setDraggingId] = useState<string | null>(null);

  const persistOrder = async (next: SettingCategoryDTO[]) => {
    onOrderChange(next);
    try {
      await saveCategoryOrder(next);
    } catch {
      /* 排序失败时 UI 仍保留本地顺序，下次 load 会恢复 */
    }
  };

  const handleDrop = (targetId: string) => {
    if (!draggingId || draggingId === targetId) return;
    const next = reorderCategories(categories, draggingId, targetId);
    setDraggingId(null);
    void persistOrder(next);
  };

  return (
    <nav className="space-y-0.5">
      {categories.map((cat) => (
        <CategoryRow
          key={cat.id}
          cat={cat}
          count={wikiCounts[cat.id] ?? 0}
          active={wikiTabActive && activeCategory === cat.id && !wikiThemeOnly}
          onSelect={() => onSelectCategory(cat.id)}
          onCreateSetting={() => onCreateSetting(cat.id)}
          onRename={() => onRenameCategory(cat.id)}
          onDelete={() => onDeleteCategory(cat)}
          onPin={() => void persistOrder(pinCategory(categories, cat.id))}
          onDragStart={() => setDraggingId(cat.id)}
          onDragOver={(e) => {
            e.preventDefault();
          }}
          onDrop={(e) => {
            e.preventDefault();
            handleDrop(cat.id);
          }}
        />
      ))}
    </nav>
  );
}
