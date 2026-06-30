import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

type Props = {
  content: string;
  className?: string;
  paper?: boolean;
};

export default function MarkdownPreview({ content, className = "", paper = false }: Props) {
  return (
    <div
      className={`markdown-prose ${paper ? "markdown-prose-paper" : ""} ${className}`}
    >
      <ReactMarkdown remarkPlugins={[remarkGfm]}>{content || ""}</ReactMarkdown>
    </div>
  );
}
