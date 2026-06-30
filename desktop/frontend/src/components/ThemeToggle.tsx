import { Moon, Sun } from "lucide-react";
import { useTheme } from "../context/ThemeContext";

type Props = {
  className?: string;
  showLabel?: boolean;
};

export default function ThemeToggle({ className = "", showLabel = false }: Props) {
  const { theme, toggleTheme } = useTheme();
  const isDark = theme === "dark";

  return (
    <button
      type="button"
      onClick={toggleTheme}
      className={`inline-flex items-center gap-2 rounded-lg border border-studio-border px-3 py-2 text-sm text-studio-muted transition hover:border-studio-muted hover:bg-studio-panel hover:text-studio-text ${className}`}
      title={isDark ? "切换为浅色模式" : "切换为深色模式"}
      aria-label={isDark ? "切换为浅色模式" : "切换为深色模式"}
    >
      {isDark ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
      {showLabel && <span>{isDark ? "浅色" : "深色"}</span>}
    </button>
  );
}
