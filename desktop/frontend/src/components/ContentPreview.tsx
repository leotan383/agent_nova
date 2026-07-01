import MarkdownPreview from "./MarkdownPreview";
import StructuredContentPreview from "./StructuredContentPreview";
import { tryParseJSON } from "../lib/structuredContent";

type Props = {
  content: string;
  className?: string;
  paper?: boolean;
  relaxed?: boolean;
};

export default function ContentPreview({ content, className = "", paper = false, relaxed = false }: Props) {
  const trimmed = content.trim();
  if (!trimmed) return null;

  const isJSON = tryParseJSON(trimmed) !== null;

  if (isJSON) {
    return <StructuredContentPreview content={trimmed} className={className} />;
  }

  return <MarkdownPreview content={trimmed} paper={paper} relaxed={relaxed} className={className} />;
}
