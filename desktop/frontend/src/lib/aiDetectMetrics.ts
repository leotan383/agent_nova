import { ChapterAIDetectMetricsDTO } from "./wails";

export type AIDetectMetrics = {
  aiScore: number | null;
  humanScore: number | null;
  riskLevel: string;
  publishable: boolean | null;
  signals: string[];
  hotspots: { excerpt: string; reason: string; fix: string }[];
};

export function riskLevelLabel(level: string): string {
  switch (level.trim().toLowerCase()) {
    case "low":
      return "低风险";
    case "medium":
      return "中风险";
    case "high":
      return "高风险";
    default:
      return "待评估";
  }
}

export function riskLevelColor(level: string): string {
  switch (level.trim().toLowerCase()) {
    case "low":
      return "text-[rgb(var(--studio-diff-add-stat))]";
    case "medium":
      return "text-[rgb(var(--studio-warning-fg))]";
    case "high":
      return "text-[rgb(var(--studio-danger-fg))]";
    default:
      return "text-studio-muted";
  }
}

export function aiScoreColor(score: number): string {
  if (score <= 3) return "text-[rgb(var(--studio-diff-add-stat))]";
  if (score <= 6) return "text-[rgb(var(--studio-warning-fg))]";
  return "text-[rgb(var(--studio-danger-fg))]";
}

/** 从 AI味.md 末尾 JSON 块解析指标 */
export function parseAIDetectMetricsFromText(content: string): AIDetectMetrics | null {
  let s = content.trim();
  const fenced = s.match(/```(?:json)?\s*\n([\s\S]*?)```\s*$/i);
  if (fenced) {
    s = fenced[1].trim();
  } else {
    const idx = s.lastIndexOf('"ai_score"');
    if (idx >= 0) {
      const start = s.lastIndexOf("{", idx);
      const end = s.lastIndexOf("}");
      if (start >= 0 && end > start) s = s.slice(start, end + 1);
    }
  }
  if (!s.includes("ai_score")) return null;
  try {
    const parsed = JSON.parse(s) as {
      ai_score?: number;
      human_score?: number;
      risk_level?: string;
      publishable?: boolean;
      signals?: string[];
      hotspots?: { excerpt?: string; reason?: string; fix?: string }[];
    };
    return {
      aiScore: typeof parsed.ai_score === "number" ? parsed.ai_score : null,
      humanScore: typeof parsed.human_score === "number" ? parsed.human_score : null,
      riskLevel: parsed.risk_level ?? "",
      publishable: typeof parsed.publishable === "boolean" ? parsed.publishable : null,
      signals: Array.isArray(parsed.signals) ? parsed.signals : [],
      hotspots: Array.isArray(parsed.hotspots)
        ? parsed.hotspots.map((h) => ({
            excerpt: h.excerpt ?? "",
            reason: h.reason ?? "",
            fix: h.fix ?? "",
          }))
        : [],
    };
  } catch {
    return null;
  }
}

export function metricsFromAIDetectDTO(dto: ChapterAIDetectMetricsDTO | null): AIDetectMetrics | null {
  if (!dto?.exists) return null;
  return {
    aiScore: Number.isFinite(dto.ai_score) ? dto.ai_score : null,
    humanScore: Number.isFinite(dto.human_score) ? dto.human_score : null,
    riskLevel: dto.risk_level ?? "",
    publishable: dto.publishable,
    signals: dto.signals ?? [],
    hotspots: (dto.hotspots ?? []).map((h) => ({
      excerpt: h.excerpt,
      reason: h.reason,
      fix: h.fix,
    })),
  };
}

export function mergeAIDetectMetrics(
  fromAPI: ChapterAIDetectMetricsDTO | null,
  fromText: AIDetectMetrics | null,
): AIDetectMetrics | null {
  const api = metricsFromAIDetectDTO(fromAPI);
  if (api) return api;
  return fromText;
}
