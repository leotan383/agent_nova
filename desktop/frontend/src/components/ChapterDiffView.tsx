import { DiffResultDTO } from "../lib/wails";

type Props = {
  diff: DiffResultDTO;
  maxHeight?: string;
  className?: string;
};

export default function ChapterDiffView({ diff, maxHeight = "max-h-96", className = "" }: Props) {
  const hasChanges = diff.lines.some((l) => l.type !== "same");

  return (
    <div className={className}>
      <div className="mb-2 flex flex-wrap items-center gap-3 text-xs text-studio-muted">
        <span>
          {diff.from_label} → {diff.to_label}
        </span>
        {(diff.added_words > 0 || diff.removed_words > 0) && (
          <span className="tabular-nums">
            {diff.added_words > 0 && (
              <span className="text-[rgb(var(--studio-diff-add-stat))]">+{diff.added_words} 字</span>
            )}
            {diff.added_words > 0 && diff.removed_words > 0 && " · "}
            {diff.removed_words > 0 && (
              <span className="text-[rgb(var(--studio-diff-del-stat))]">−{diff.removed_words} 字</span>
            )}
          </span>
        )}
      </div>
      {!hasChanges ? (
        <p className="rounded-lg border border-studio-border bg-studio-bg px-3 py-4 text-center text-sm text-studio-muted">
          无变更
        </p>
      ) : (
        <pre
          className={`overflow-y-auto whitespace-pre-wrap rounded-lg border border-studio-border bg-studio-bg font-serif text-xs leading-relaxed ${maxHeight}`}
        >
          {diff.lines.map((line, i) => {
            if (line.type === "same") {
              return (
                <span key={i} className="text-studio-ink/80">
                  {line.text}
                  {"\n"}
                </span>
              );
            }
            if (line.type === "add") {
              return (
                <span key={i} className="block studio-diff-add">
                  + {line.text}
                </span>
              );
            }
            return (
              <span key={i} className="block studio-diff-del">
                − {line.text}
              </span>
            );
          })}
        </pre>
      )}
    </div>
  );
}
