import { useCallback, useEffect, useState } from "react";
import { Loader2 } from "lucide-react";
import { ChapterDocDTO, app } from "../lib/wails";
import MarkdownEditor from "./MarkdownEditor";

type DocKind = "body" | "review" | "summary";

const TABS: { kind: DocKind; label: string }[] = [
  { kind: "body", label: "正文" },
  { kind: "review", label: "审查" },
  { kind: "summary", label: "摘要" },
];

type Props = {
  chapter: number;
  initialTab?: DocKind;
  onSaved?: () => void;
};

export default function ChapterDocumentPanel({ chapter, initialTab = "body", onSaved }: Props) {
  const [tab, setTab] = useState<DocKind>(initialTab);
  const [docs, setDocs] = useState<Record<DocKind, ChapterDocDTO | null>>({
    body: null,
    review: null,
    summary: null,
  });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  const loadAll = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [body, review, summary] = await Promise.all([
        app().GetChapterDocument(chapter, "body"),
        app().GetChapterDocument(chapter, "review"),
        app().GetChapterDocument(chapter, "summary"),
      ]);
      setDocs({ body, review, summary });
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [chapter]);

  useEffect(() => {
    setTab(initialTab);
    loadAll();
  }, [chapter, initialTab, loadAll]);

  const current = docs[tab];

  const save = async (body: string) => {
    setSaving(true);
    try {
      await app().SaveChapterDocument(chapter, tab, body);
      await loadAll();
      onSaved?.();
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="flex flex-1 items-center justify-center text-studio-muted">
        <Loader2 className="h-6 w-6 animate-spin" />
      </div>
    );
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
      <div className="flex shrink-0 gap-1 border-b border-studio-border px-3 py-2">
        {TABS.map(({ kind, label }) => {
          const exists = docs[kind]?.exists;
          return (
            <button
              key={kind}
              type="button"
              onClick={() => setTab(kind)}
              className={`rounded-lg px-3 py-1.5 text-xs transition ${
                tab === kind
                  ? "bg-studio-accent/15 text-studio-accent"
                  : "text-studio-muted hover:bg-studio-bg hover:text-studio-text"
              }`}
            >
              {label}
              {!exists && kind !== "body" && (
                <span className="ml-1 text-studio-muted/50">·</span>
              )}
            </button>
          );
        })}
      </div>

      {error && <div className="mx-3 mt-2 shrink-0 studio-alert-error-compact">{error}</div>}

      {tab !== "body" && current && !current.exists && (
        <p className="shrink-0 px-4 py-2 text-xs text-studio-muted">
          暂无{tab === "review" ? "审查报告" : "摘要"}，可在编辑模式下新建并保存。
        </p>
      )}

      <MarkdownEditor
        key={`${chapter}-${tab}-${current?.body?.length ?? 0}`}
        value={current?.body ?? ""}
        paper
        saving={saving}
        onSave={save}
        selectionChapter={tab === "body" ? chapter : undefined}
        emptyHint={
          tab === "body"
            ? "正文为空"
            : tab === "review"
              ? "暂无审查报告"
              : "暂无摘要"
        }
      />
    </div>
  );
}
