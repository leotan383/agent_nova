export type MemorySummaryItem = {
  type?: string;
  category?: string;
  subject?: string;
  content?: string;
  priority?: string;
};

export type StoryFacts = {
  entities?: Array<{ type?: string; name?: string; state?: Record<string, unknown> }>;
  foreshadows?: Array<{ id?: string; description?: string; status?: string }>;
  cool_points?: Array<{ type?: string; description?: string; delivered?: boolean }>;
  memories?: MemorySummaryItem[];
};

export function tryParseJSON(text: string): unknown | null {
  let s = text.trim();
  if (!s) return null;

  const fenced = s.match(/^```(?:json)?\s*\n?([\s\S]*?)```$/i);
  if (fenced) {
    s = fenced[1].trim();
  }

  if (!s.startsWith("{") && !s.startsWith("[")) {
    return null;
  }

  try {
    return JSON.parse(s) as unknown;
  } catch {
    return null;
  }
}

export function isMemorySummaryArray(data: unknown): data is MemorySummaryItem[] {
  if (!Array.isArray(data) || data.length === 0) return false;
  return data.every(
    (item) =>
      item &&
      typeof item === "object" &&
      ("content" in item || "subject" in item) &&
      ("type" in item || "category" in item || "priority" in item),
  );
}

export function isStoryFacts(data: unknown): data is StoryFacts {
  if (!data || typeof data !== "object" || Array.isArray(data)) return false;
  const o = data as StoryFacts;
  return !!(o.entities || o.foreshadows || o.cool_points || o.memories);
}

export const typeLabel: Record<string, string> = {
  style: "写法",
  plot: "剧情",
  character: "角色",
  world: "世界观",
};

export const priorityLabel: Record<string, string> = {
  high: "高",
  medium: "中",
  low: "低",
};
