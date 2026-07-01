const statusMeta: Record<string, { label: string; className: string }> = {
  draft: {
    label: "待审",
    className: "bg-[rgb(var(--studio-warning-bg))] text-[rgb(var(--studio-warning-fg))]",
  },
  reviewed: {
    label: "已审",
    className: "bg-[rgb(var(--studio-diff-add-bg))] text-[rgb(var(--studio-diff-add-stat))]",
  },
  published: {
    label: "已发布",
    className: "bg-studio-accent/15 text-studio-accent",
  },
};

type Props = {
  status: string;
  compact?: boolean;
};

export default function ChapterStatusBadge({ status, compact }: Props) {
  const key = status.trim().toLowerCase();
  const meta = statusMeta[key];
  if (!meta) return null;

  return (
    <span
      className={`inline-flex shrink-0 items-center rounded-full font-medium ${meta.className} ${
        compact ? "px-1.5 py-0.5 text-[10px]" : "px-2 py-0.5 text-[10px]"
      }`}
    >
      {meta.label}
    </span>
  );
}
