import { ChapterReviewMetricsDTO } from "./wails";

export type ReviewMetrics = {
  hookScore: number | null;
  coolPoint: string;
  debt: string;
  issues: string[];
};

/** 将审查 JSON 中的 issue 条目统一为展示用字符串（兼容对象格式）。 */
export function normalizeReviewIssue(item: unknown): string {
  if (typeof item === "string") return item.trim();
  if (!item || typeof item !== "object") return String(item ?? "");
  const o = item as Record<string, unknown>;
  const position = o.position ?? o["位置"];
  const problem = o.problem ?? o["问题"];
  const suggestion = o.suggestion ?? o["建议"];
  const parts: string[] = [];
  if (position != null && String(position).trim()) parts.push(String(position).trim());
  if (problem != null && String(problem).trim()) parts.push(String(problem).trim());
  if (suggestion != null && String(suggestion).trim()) {
    parts.push(`建议：${String(suggestion).trim()}`);
  }
  if (parts.length > 0) return parts.join(" · ");
  try {
    return JSON.stringify(item);
  } catch {
    return String(item);
  }
}

function normalizeReviewIssues(raw: unknown): string[] {
  if (!Array.isArray(raw)) return [];
  return raw.map(normalizeReviewIssue).filter((s) => s.length > 0);
}

/** 从审查 Markdown 末尾 JSON 块解析指标（数据库无记录时的兜底） */
export function parseReviewMetricsFromText(content: string): ReviewMetrics | null {
  let s = content.trim();
  const fenced = s.match(/```(?:json)?\s*\n([\s\S]*?)```\s*$/i);
  if (fenced) {
    s = fenced[1].trim();
  } else {
    const hookIdx = s.lastIndexOf('"hook_score"');
    if (hookIdx >= 0) {
      const start = s.lastIndexOf("{", hookIdx);
      const end = s.lastIndexOf("}");
      if (start >= 0 && end > start) s = s.slice(start, end + 1);
    }
  }
  if (!s.includes("hook_score")) return null;
  try {
    const parsed = JSON.parse(s) as {
      hook_score?: number;
      cool_point?: string;
      debt?: string;
      issues?: unknown[];
    };
    return {
      hookScore: typeof parsed.hook_score === "number" ? parsed.hook_score : null,
      coolPoint: parsed.cool_point ?? "",
      debt: parsed.debt ?? "",
      issues: normalizeReviewIssues(parsed.issues),
    };
  } catch {
    return null;
  }
}

export function metricsFromDTO(dto: ChapterReviewMetricsDTO | null): ReviewMetrics | null {
  if (!dto?.exists) return null;
  return {
    hookScore: dto.hook_score,
    coolPoint: dto.cool_point ?? "",
    debt: dto.debt ?? "",
    issues: normalizeReviewIssues(dto.issues),
  };
}

export function mergeReviewMetrics(
  fromDB: ChapterReviewMetricsDTO | null,
  fromText: ReviewMetrics | null,
): ReviewMetrics | null {
  const db = metricsFromDTO(fromDB);
  if (db) {
    if ((!db.issues || db.issues.length === 0) && fromText?.issues?.length) {
      return { ...db, issues: fromText.issues };
    }
    return db;
  }
  return fromText;
}
