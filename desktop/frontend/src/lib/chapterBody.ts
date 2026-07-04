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

const chapterHeadingRe =
  /^(?:#{1,6}\s*)?第\s*[0-9一二三四五六七八九十百千万零〇两\d]+\s*章/;

const preambleKeywords = [
  "以下为",
  "以下是",
  "润色版本",
  "润色说明",
  "修改说明",
  "修订说明",
  "重点调整",
  "审查意见",
  "根据审查",
  "修改的润色",
];

function isPreambleLine(line: string): boolean {
  return preambleKeywords.some((k) => line.includes(k));
}

/** 去掉审查润色时误留在正文开头的说明段，定位到章节标题或 --- 后的正文 */
export function trimChapterBodyPreamble(content: string): string {
  let s = content.replace(/\r\n/g, "\n");
  const lines = s.split("\n");
  let pos = 0;

  for (let i = 0; i < lines.length; i++) {
    const trimmed = lines[i].trim();
    if (!trimmed) {
      pos += lines[i].length + 1;
      continue;
    }
    if (chapterHeadingRe.test(trimmed)) {
      if (pos > 0) return s.slice(pos).trimStart();
      return s;
    }
    if (isPreambleLine(trimmed)) {
      pos += lines[i].length + 1;
      continue;
    }
    if (trimmed === "---" || trimmed === "***") {
      pos += lines[i].length + 1;
      for (let j = i + 1; j < lines.length; j++) {
        const next = lines[j].trim();
        if (!next) continue;
        if (chapterHeadingRe.test(next)) {
          let offset = pos;
          for (let k = i + 1; k < j; k++) offset += lines[k].length + 1;
          return s.slice(offset).trimStart();
        }
        break;
      }
      continue;
    }
    return s;
  }
  return s;
}

/** 去掉正文末尾误写入的润色说明、修改对照表等审查附录 */
export function trimChapterBodyAppendix(content: string): string {
  const lines = content.replace(/\r\n/g, "\n").split("\n");
  const appendixHeadingRe =
    /^#{1,6}\s*(润色说明|修改说明|修订说明|润色记录|修改记录|修改对照)\s*$/;
  const boldAppendix = ["润色说明", "修改说明", "修订说明", "修改对照"];

  let cutLine = -1;
  for (let i = 0; i < lines.length; i++) {
    const trimmed = lines[i].trim();
    if (
      appendixHeadingRe.test(trimmed) ||
      boldAppendix.some((k) => trimmed === `**${k}**`)
    ) {
      cutLine = i;
      break;
    }
  }
  if (cutLine < 0) return content;

  let start = cutLine;
  while (start > 0 && lines[start - 1].trim() === "") start--;
  if (start > 0 && lines[start - 1].trim() === "---") start--;
  while (start > 0 && lines[start - 1].trim() === "") start--;

  return lines.slice(0, start).join("\n").trimEnd();
}

/** 展示/编辑用：去掉审查 JSON 尾缀、开头说明段与末尾润色说明 */
export function normalizeChapterBodyForDisplay(content: string): string {
  return trimChapterBodyAppendix(
    trimChapterBodyPreamble(stripReviewMetricsSuffix(content)),
  );
}
