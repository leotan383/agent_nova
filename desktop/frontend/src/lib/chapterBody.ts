/** 去掉写章/审查流水线误写入正文末尾的 hook_score 等 JSON 块 */
export function stripReviewMetricsSuffix(content: string): string {
  let s = content.replace(/\r\n/g, "\n").trimEnd();

  const fenced = s.match(/\n```(?:json)?\s*\n[\s\S]*?```\s*$/i);
  if (fenced && /hook_score/i.test(fenced[0])) {
    s = s.slice(0, fenced.index).trimEnd();
  }

  const hookIdx = s.lastIndexOf('"hook_score"');
  if (hookIdx >= 0) {
    const objStart = s.lastIndexOf("{", hookIdx);
    const objEnd = s.lastIndexOf("}");
    if (objStart >= 0 && objEnd > objStart) {
      const candidate = s.slice(objStart, objEnd + 1);
      try {
        const parsed = JSON.parse(candidate) as Record<string, unknown>;
        if ("hook_score" in parsed) {
          s = s.slice(0, objStart).trimEnd();
        }
      } catch {
        // not valid review metrics JSON
      }
    }
  }

  return s;
}
