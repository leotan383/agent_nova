import {
  BookOpen,
  LayoutGrid,
  MapPin,
  Package,
  ScrollText,
  Swords,
  Users,
  type LucideIcon,
} from "lucide-react";

const icons: Record<string, LucideIcon> = {
  角色: Users,
  世界观: BookOpen,
  势力: Swords,
  地点: MapPin,
  物品: Package,
  其他: ScrollText,
};

export function settingCategoryIcon(categoryId: string): LucideIcon {
  return icons[categoryId] ?? LayoutGrid;
}
