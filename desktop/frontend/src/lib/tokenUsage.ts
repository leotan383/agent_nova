import { TokenUsageDTO, ProjectTokenUsageDTO } from "./wails";

/** 格式化美元成本，小金额保留足够精度 */
export function formatCostUSD(usd: number): string {
  if (usd <= 0) return "";
  if (usd >= 0.01) return `≈ $${usd.toFixed(2)}`;
  if (usd >= 0.001) return `≈ $${usd.toFixed(3)}`;
  return `≈ $${usd.toFixed(4)}`;
}

/** 格式化 token 用量为一行可读文本 */
export function formatTokenUsage(u: TokenUsageDTO | ProjectTokenUsageDTO): string {
  const total = u.total_tokens ?? u.prompt_tokens + u.completion_tokens;
  const parts = [`约 ${total.toLocaleString("zh-CN")} tokens`];
  if ("write_runs" in u && u.write_runs > 0) {
    parts.push(`${u.write_runs} 次写章`);
  }
  const cost = u.estimated_cost_usd ?? 0;
  const costStr = formatCostUSD(cost);
  if (costStr) parts.push(costStr);
  return parts.join(" · ");
}
