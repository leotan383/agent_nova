import { useCallback, useEffect, useState } from "react";
import {
  CheckCircle2,
  ChevronRight,
  Circle,
  KeyRound,
  Map,
  PenLine,
  Sparkles,
  X,
} from "lucide-react";
import { ProjectHealthDTO, StatusReport, app } from "../lib/wails";

type Props = {
  novelId: string;
  status: StatusReport;
  onOpenSettings: () => void;
  onOpenPlanning: (volume?: number) => void;
  onOpenWrite: () => void;
};

type Step = {
  id: string;
  title: string;
  detail: string;
  done: boolean;
  actionLabel: string;
  onAction: () => void;
  icon: typeof KeyRound;
};

function onboardingKey(novelId: string) {
  return `nova-onboarding-dismissed-${novelId}`;
}

export function isOnboardingDismissed(novelId: string) {
  try {
    return localStorage.getItem(onboardingKey(novelId)) === "1";
  } catch {
    return false;
  }
}

export function dismissOnboarding(novelId: string) {
  try {
    localStorage.setItem(onboardingKey(novelId), "1");
  } catch {
    /* ignore */
  }
}

export default function OnboardingChecklist({
  novelId,
  status,
  onOpenSettings,
  onOpenPlanning,
  onOpenWrite,
}: Props) {
  const [hasKey, setHasKey] = useState(true);
  const [health, setHealth] = useState<ProjectHealthDTO | null>(null);
  const [dismissed, setDismissed] = useState(() => isOnboardingDismissed(novelId));

  const load = useCallback(async () => {
    try {
      const [keyOk, h] = await Promise.all([app().HasAPIKey(), app().GetProjectHealth()]);
      setHasKey(keyOk);
      setHealth(h);
    } catch {
      setHasKey(false);
      setHealth(null);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const hasOutline = health?.has_volume_outline ?? false;
  const hasChapter = status.chapter_count > 0;
  const allDone = hasKey && hasOutline && hasChapter;

  if (dismissed || allDone) return null;

  const steps: Step[] = [
    {
      id: "api-key",
      title: "配置 API Key",
      detail: "在右上角设置中填写 OpenAI 兼容 API，写章与规划才能调用 AI。",
      done: hasKey,
      actionLabel: "去设置",
      onAction: onOpenSettings,
      icon: KeyRound,
    },
    {
      id: "plan",
      title: "规划第 1 卷卷纲",
      detail: "生成章纲后，AI 写章时会自动提取每章冲突、爽点与伏笔任务。",
      done: hasOutline,
      actionLabel: "去规划",
      onAction: () => onOpenPlanning(Math.max(1, status.current_volume)),
      icon: Map,
    },
    {
      id: "write",
      title: "写出第 1 章",
      detail: "完成写前检查后，AI 将依次起草、润色、摘要并沉淀记忆。",
      done: hasChapter,
      actionLabel: "去写章",
      onAction: onOpenWrite,
      icon: PenLine,
    },
  ];

  const doneCount = steps.filter((s) => s.done).length;
  const nextStep = steps.find((s) => !s.done);

  const handleDismiss = () => {
    dismissOnboarding(novelId);
    setDismissed(true);
  };

  return (
    <section className="overflow-hidden rounded-2xl border border-studio-accent/25 bg-gradient-to-br from-studio-accent/8 to-transparent shadow-sm">
      <div className="flex items-start justify-between gap-3 border-b border-studio-accent/15 px-5 py-4">
        <div className="flex items-start gap-3">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-studio-accent/15">
            <Sparkles className="h-4 w-4 text-studio-accent" />
          </div>
          <div>
            <h3 className="text-sm font-semibold text-studio-text">新手引导 · 30 分钟出第一章</h3>
            <p className="mt-0.5 text-xs text-studio-muted">
              已完成 {doneCount}/{steps.length} 步
              {nextStep && ` · 下一步：${nextStep.title}`}
            </p>
          </div>
        </div>
        <button
          type="button"
          onClick={handleDismiss}
          className="rounded-lg p-1.5 text-studio-muted hover:bg-studio-panel hover:text-studio-text"
          title="不再显示"
        >
          <X className="h-4 w-4" />
        </button>
      </div>

      <ol className="space-y-0 divide-y divide-studio-border/50 px-5">
        {steps.map((step, idx) => {
          const Icon = step.icon;
          return (
            <li key={step.id} className="flex items-start gap-3 py-3.5">
              <div className="mt-0.5 shrink-0">
                {step.done ? (
                  <CheckCircle2 className="h-5 w-5 text-[rgb(var(--studio-diff-add-stat))]" />
                ) : (
                  <Circle className="h-5 w-5 text-studio-muted/40" />
                )}
              </div>
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-[10px] font-medium uppercase tracking-wide text-studio-muted">
                    步骤 {idx + 1}
                  </span>
                  <Icon className="h-3.5 w-3.5 text-studio-muted" />
                  <p className={`text-sm font-medium ${step.done ? "text-studio-muted line-through" : "text-studio-text"}`}>
                    {step.title}
                  </p>
                </div>
                <p className="mt-1 text-xs leading-relaxed text-studio-muted">{step.detail}</p>
                {!step.done && (
                  <button
                    type="button"
                    onClick={step.onAction}
                    className="mt-2 inline-flex items-center gap-1 rounded-lg bg-studio-accent/10 px-3 py-1.5 text-xs font-medium text-studio-accent hover:bg-studio-accent/15"
                  >
                    {step.actionLabel}
                    <ChevronRight className="h-3.5 w-3.5" />
                  </button>
                )}
              </div>
            </li>
          );
        })}
      </ol>
    </section>
  );
}
