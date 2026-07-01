import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

type Props = {
  content: string;
  className?: string;
  paper?: boolean;
  relaxed?: boolean;
};

export default function MarkdownPreview({ content, className = "", paper = false, relaxed = false }: Props) {
  return (
    <div
      className={`markdown-prose ${paper ? "markdown-prose-paper" : ""} ${relaxed ? "markdown-prose-relaxed" : ""} ${className}`}
    >
      <ReactMarkdown remarkPlugins={[remarkGfm]}>{content || ""}</ReactMarkdown>
    </div>
  );
}
