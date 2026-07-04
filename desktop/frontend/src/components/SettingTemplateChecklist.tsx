import { useCallback, useEffect, useState } from "react";
import { CheckCircle2, Circle, Loader2 } from "lucide-react";
import { SettingChecklistDTO, fetchSettingChecklist } from "../lib/settingCategories";
import { SettingCategory } from "../lib/wikiCategories";

type Props = {
  refreshKey?: number;
  categoryFilter?: SettingCategory | null;
  onCreateItem: (category: string, title: string, templateKind: string) => void;
  onOpenSetting?: (settingRel: string) => void;
  className?: string;
};

export default function SettingTemplateChecklist({
  refreshKey = 0,
  categoryFilter = null,
  onCreateItem,
  onOpenSetting,
  className = "mx-1 mb-2",
}: Props) {
  const [checklist, setChecklist] = useState<SettingChecklistDTO | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await fetchSettingChecklist();
      setChecklist(data);
    } catch {
      setChecklist(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load, refreshKey]);

  if (loading) {
    return (
      <div className="flex items-center gap-2 px-3 py-2 text-xs text-studio-muted">
        <Loader2 className="h-3.5 w-3.5 animate-spin" />
        加载模板清单…
      </div>
    );
  }

  if (!checklist || checklist.total === 0) return null;

  const items = categoryFilter
    ? checklist.items.filter((it) => it.category_id === categoryFilter)
    : checklist.items.filter((it) => !it.done);

  if (items.length === 0) return null;

  const title = categoryFilter
    ? "推荐补全"
    : `${checklist.genre} 模板 · ${checklist.done_count}/${checklist.total}`;

  return (
    <div className={`${className} rounded-lg border border-studio-border/70 bg-studio-bg/40 px-2 py-2`}>
      <p className="mb-1.5 px-1 text-[10px] font-medium text-studio-muted">{title}</p>
      <ul className="space-y-1">
        {items.map((it) => (
          <li key={it.id}>
            {it.done ? (
              <button
                type="button"
                onClick={() => it.setting_rel && onOpenSetting?.(`setting:${it.setting_rel}`)}
                className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-xs text-studio-muted hover:bg-studio-panel hover:text-studio-text"
              >
                <CheckCircle2 className="h-3.5 w-3.5 shrink-0 text-[rgb(var(--studio-diff-add-stat))]" />
                <span className="truncate">{it.title}</span>
              </button>
            ) : (
              <button
                type="button"
                onClick={() => onCreateItem(it.category_id, it.title, it.template_kind)}
                className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-xs text-studio-text hover:bg-studio-panel"
              >
                <Circle className="h-3.5 w-3.5 shrink-0 text-studio-muted" />
                <span className="min-w-0 flex-1 truncate">{it.title}</span>
                <span className="shrink-0 text-[10px] text-studio-accent">创建</span>
              </button>
            )}
          </li>
        ))}
      </ul>
    </div>
  );
}
