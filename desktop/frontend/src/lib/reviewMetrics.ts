import { ChapterReviewMetricsDTO } from "./wails";

export type ReviewMetrics = {
  hookScore: number | null;
  coolPoint: string;
  debt: string;
  issues: string[];
};

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
      issues?: string[];
    };
    return {
      hookScore: typeof parsed.hook_score === "number" ? parsed.hook_score : null,
      coolPoint: parsed.cool_point ?? "",
      debt: parsed.debt ?? "",
      issues: Array.isArray(parsed.issues) ? parsed.issues : [],
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
    issues: dto.issues ?? [],
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
