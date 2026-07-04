type ReportSubview = "summary" | "full";

type Props = {
  value: ReportSubview;
  onChange: (value: ReportSubview) => void;
  accent?: "default" | "ai";
};

export type { ReportSubview };

export default function ReportSubTabBar({ value, onChange, accent = "default" }: Props) {
  const activeClass =
    accent === "ai"
      ? "bg-studio-ai/15 text-studio-ai"
      : "bg-studio-accent/15 text-studio-accent";

  const tabs: { id: ReportSubview; label: string }[] = [
    { id: "summary", label: "摘要" },
    { id: "full", label: "完整报告" },
  ];

  return (
    <div className="flex shrink-0 gap-1 border-b border-studio-border bg-studio-panel/30 px-3 py-1.5">
      {tabs.map(({ id, label }) => (
        <button
          key={id}
          type="button"
          onClick={() => onChange(id)}
          className={`rounded-md px-3 py-1 text-xs font-medium transition ${
            value === id ? activeClass : "text-studio-muted hover:bg-studio-bg hover:text-studio-text"
          }`}
        >
          {label}
        </button>
      ))}
    </div>
  );
}
