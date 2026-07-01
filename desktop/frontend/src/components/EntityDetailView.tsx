import { Boxes, MapPin, Package, User } from "lucide-react";
import { EntityDTO } from "../lib/wails";

const typeMeta: Record<
  string,
  { label: string; icon: typeof User; accent: string; chip: string }
> = {
  character: {
    label: "人物",
    icon: User,
    accent: "text-studio-accent",
    chip: "bg-studio-accent/10 text-studio-accent",
  },
  location: {
    label: "地点",
    icon: MapPin,
    accent: "text-[rgb(var(--studio-diff-add-stat))]",
    chip: "bg-[rgb(var(--studio-diff-add-bg))] text-[rgb(var(--studio-diff-add-stat))]",
  },
  item: {
    label: "物品",
    icon: Package,
    accent: "text-studio-ai",
    chip: "bg-studio-ai/10 text-studio-ai",
  },
};

export function metaFor(type: string) {
  return (
    typeMeta[type] ?? {
      label: type,
      icon: Boxes,
      accent: "text-studio-muted",
      chip: "bg-studio-border/40 text-studio-muted",
    }
  );
}

export default function EntityDetailView({ entity }: { entity: EntityDTO }) {
  const meta = metaFor(entity.type);
  const Icon = meta.icon;
  const stateEntries = Object.entries(entity.state);
  const initial = entity.name.trim().charAt(0) || "?";

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
      <div className="shrink-0 border-b border-studio-border bg-gradient-to-br from-studio-bg/40 to-transparent px-6 py-5">
        <div className="flex flex-wrap items-start gap-4">
          <div
            className={`flex h-14 w-14 shrink-0 items-center justify-center rounded-2xl text-xl font-semibold ${meta.chip}`}
          >
            {entity.type === "character" ? initial : <Icon className="h-6 w-6" />}
          </div>
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center gap-2">
              <h3 className="text-xl font-semibold text-studio-text">{entity.name}</h3>
              <span className={`rounded-full px-2.5 py-0.5 text-[10px] font-medium ${meta.chip}`}>
                {meta.label}
              </span>
              <span className="rounded-full bg-studio-bg px-2 py-0.5 text-[10px] text-studio-muted">
                审查自动更新
              </span>
            </div>
            {entity.last_chapter > 0 && (
              <p className="mt-1.5 text-sm text-studio-muted">最近更新于第 {entity.last_chapter} 章</p>
            )}
          </div>
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto px-6 py-5">
        {stateEntries.length > 0 ? (
          <>
            <h4 className="mb-3 text-xs font-medium uppercase tracking-wide text-studio-muted">当前状态</h4>
            <dl className="grid gap-3 sm:grid-cols-2">
              {stateEntries.map(([k, v]) => (
                <div
                  key={k}
                  className="rounded-xl border border-studio-border bg-studio-bg/50 px-4 py-3 transition hover:border-studio-border hover:bg-studio-bg"
                >
                  <dt className="text-[10px] font-medium uppercase tracking-wide text-studio-muted">{k}</dt>
                  <dd className="mt-1.5 text-sm leading-relaxed text-studio-text">{v}</dd>
                </div>
              ))}
            </dl>
          </>
        ) : (
          <div className="flex flex-col items-center justify-center py-12 text-center">
            <p className="text-sm text-studio-muted">暂无结构化状态字段</p>
            <p className="mt-1 max-w-xs text-xs text-studio-muted/70">
              后续章节审查时，若正文涉及该实体，会自动补充状态信息。
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
