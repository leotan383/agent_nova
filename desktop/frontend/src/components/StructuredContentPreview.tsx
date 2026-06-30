import {
  MemorySummaryItem,
  StoryFacts,
  isMemorySummaryArray,
  isStoryFacts,
  priorityLabel,
  tryParseJSON,
  typeLabel,
} from "../lib/structuredContent";

type Props = {
  content: string;
  className?: string;
};

function PriorityBadge({ priority }: { priority?: string }) {
  if (!priority) return null;
  const label = priorityLabel[priority] || priority;
  const cls =
    priority === "high"
      ? "bg-[rgb(var(--studio-diff-del-stat)/0.15)] text-[rgb(var(--studio-diff-del-stat))]"
      : priority === "medium"
        ? "bg-[rgb(var(--studio-warning-bg))] text-[rgb(var(--studio-warning-fg))]"
        : "bg-studio-border text-studio-muted";
  return (
    <span className={`rounded px-1.5 py-0.5 text-[10px] font-medium ${cls}`}>{label}优先级</span>
  );
}

function MemoryCards({ items }: { items: MemorySummaryItem[] }) {
  return (
    <ul className="space-y-3">
      {items.map((item, i) => {
        const kind = item.type || item.category || "note";
        return (
          <li
            key={i}
            className="rounded-lg border border-studio-border bg-studio-panel/80 p-4"
          >
            <div className="mb-2 flex flex-wrap items-center gap-2">
              <span className="rounded bg-studio-accent/15 px-2 py-0.5 text-xs font-medium text-studio-accent">
                {typeLabel[kind] || kind}
              </span>
              {item.subject && item.type && (
                <span className="text-xs text-studio-muted">{item.subject}</span>
              )}
              <PriorityBadge priority={item.priority} />
            </div>
            <p className="text-sm leading-relaxed text-studio-text">{item.content}</p>
          </li>
        );
      })}
    </ul>
  );
}

function StoryFactsView({ facts }: { facts: StoryFacts }) {
  return (
    <div className="space-y-6">
      {facts.memories && facts.memories.length > 0 && (
        <section>
          <h3 className="mb-3 text-sm font-medium text-studio-muted">记忆要点</h3>
          <MemoryCards items={facts.memories} />
        </section>
      )}
      {facts.entities && facts.entities.length > 0 && (
        <section>
          <h3 className="mb-3 text-sm font-medium text-studio-muted">实体状态</h3>
          <ul className="space-y-2">
            {facts.entities.map((e, i) => (
              <li key={i} className="rounded-lg border border-studio-border p-3 text-sm">
                <span className="font-medium">{e.name}</span>
                <span className="ml-2 text-xs text-studio-muted">{e.type}</span>
                {e.state && Object.keys(e.state).length > 0 && (
                  <pre className="mt-2 overflow-x-auto rounded bg-studio-bg p-2 text-xs text-studio-muted">
                    {JSON.stringify(e.state, null, 2)}
                  </pre>
                )}
              </li>
            ))}
          </ul>
        </section>
      )}
      {facts.foreshadows && facts.foreshadows.length > 0 && (
        <section>
          <h3 className="mb-3 text-sm font-medium text-studio-muted">伏笔</h3>
          <ul className="space-y-2">
            {facts.foreshadows.map((f, i) => (
              <li key={i} className="rounded-lg border border-studio-border p-3 text-sm">
                <p>{f.description}</p>
                <p className="mt-1 text-xs text-studio-muted">
                  {f.id} · {f.status}
                </p>
              </li>
            ))}
          </ul>
        </section>
      )}
      {facts.cool_points && facts.cool_points.length > 0 && (
        <section>
          <h3 className="mb-3 text-sm font-medium text-studio-muted">爽点</h3>
          <ul className="space-y-2">
            {facts.cool_points.map((c, i) => (
              <li key={i} className="rounded-lg border border-studio-border p-3 text-sm">
                <p>{c.description}</p>
                <p className="mt-1 text-xs text-studio-muted">
                  {c.type} · {c.delivered ? "已兑现" : "未兑现"}
                </p>
              </li>
            ))}
          </ul>
        </section>
      )}
    </div>
  );
}

function PrettyJSON({ data }: { data: unknown }) {
  return (
    <pre className="overflow-x-auto rounded-lg border border-studio-border bg-studio-bg p-4 font-mono text-xs leading-relaxed text-studio-text">
      {JSON.stringify(data, null, 2)}
    </pre>
  );
}

export default function StructuredContentPreview({ content, className = "" }: Props) {
  const parsed = tryParseJSON(content);

  if (parsed && isMemorySummaryArray(parsed)) {
    return (
      <div className={className}>
        <MemoryCards items={parsed} />
      </div>
    );
  }

  if (parsed && isStoryFacts(parsed)) {
    return (
      <div className={className}>
        <StoryFactsView facts={parsed} />
      </div>
    );
  }

  if (parsed !== null) {
    return (
      <div className={className}>
        <PrettyJSON data={parsed} />
      </div>
    );
  }

  return null;
}
