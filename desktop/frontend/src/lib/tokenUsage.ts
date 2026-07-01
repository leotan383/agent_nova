import { TokenUsageDTO, ProjectTokenUsageDTO } from "./wails";

/** 格式化 token 用量为一行可读文本 */
export function formatTokenUsage(u: TokenUsageDTO | ProjectTokenUsageDTO): string {
  const total = u.total_tokens ?? u.prompt_tokens + u.completion_tokens;
  const parts = [`约 ${total.toLocaleString("zh-CN")} tokens`];
  if ("write_runs" in u && u.write_runs > 0) {
    parts.push(`${u.write_runs} 次写章`);
  }
  if (u.estimated_cost_usd && u.estimated_cost_usd > 0) {
    parts.push(`≈ $${u.estimated_cost_usd.toFixed(3)}`);
  }
  return parts.join(" · ");
}
