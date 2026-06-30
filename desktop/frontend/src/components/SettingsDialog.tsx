import { useEffect, useState } from "react";
import { Loader2, Save, Settings, X } from "lucide-react";
import { AppConfigDTO, SaveAppConfigInput, app } from "../lib/wails";

type Props = {
  open: boolean;
  onClose: () => void;
  onSaved?: () => void;
};

export default function SettingsDialog({ open, onClose, onSaved }: Props) {
  const [config, setConfig] = useState<AppConfigDTO | null>(null);
  const [model, setModel] = useState("");
  const [baseURL, setBaseURL] = useState("");
  const [apiKey, setAPIKey] = useState("");
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  useEffect(() => {
    if (!open) return;
    setError("");
    setSuccess("");
    setAPIKey("");
    setLoading(true);
    app()
      .GetAppConfig()
      .then((c) => {
        setConfig(c);
        setModel(c.model);
        setBaseURL(c.base_url);
      })
      .catch((e) => setError(e instanceof Error ? e.message : String(e)))
      .finally(() => setLoading(false));
  }, [open]);

  const save = async () => {
    setSaving(true);
    setError("");
    setSuccess("");
    try {
      const input: SaveAppConfigInput = {
        model: model.trim(),
        base_url: baseURL.trim(),
        api_key: apiKey.trim(),
      };
      await app().SaveAppConfig(input);
      setSuccess("配置已保存");
      setAPIKey("");
      const c = await app().GetAppConfig();
      setConfig(c);
      onSaved?.();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  };

  if (!open) return null;

  return (
    <div className="studio-modal-overlay fixed inset-0 z-50 flex items-center justify-center p-4" onClick={onClose}>
      <div
        className="w-full max-w-md rounded-xl border border-studio-border bg-studio-panel shadow-card"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between border-b border-studio-border px-5 py-4">
          <div className="flex items-center gap-2">
            <Settings className="h-4 w-4 text-studio-accent" />
            <h2 className="text-base font-medium">应用设置</h2>
          </div>
          <button type="button" onClick={onClose} className="rounded p-1 text-studio-muted hover:text-studio-text">
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="space-y-4 p-5">
          {loading ? (
            <div className="flex justify-center py-8">
              <Loader2 className="h-6 w-6 animate-spin text-studio-muted" />
            </div>
          ) : (
            <>
              <div>
                <label className="mb-1 block text-xs text-studio-muted">模型</label>
                <input
                  value={model}
                  onChange={(e) => setModel(e.target.value)}
                  placeholder="gpt-4o"
                  className="w-full rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none focus:border-studio-accent"
                />
              </div>
              <div>
                <label className="mb-1 block text-xs text-studio-muted">API Base URL</label>
                <input
                  value={baseURL}
                  onChange={(e) => setBaseURL(e.target.value)}
                  placeholder="https://api.openai.com/v1"
                  className="w-full rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none focus:border-studio-accent"
                />
                <p className="mt-1 text-[10px] text-studio-muted">留空则使用 OpenAI 默认端点</p>
              </div>
              <div>
                <label className="mb-1 block text-xs text-studio-muted">API Key</label>
                <input
                  type="password"
                  value={apiKey}
                  onChange={(e) => setAPIKey(e.target.value)}
                  placeholder={config?.has_api_key ? `已配置 ${config.api_key_mask}` : "sk-..."}
                  className="w-full rounded-lg border border-studio-border bg-studio-bg px-3 py-2 text-sm outline-none focus:border-studio-accent"
                />
                <p className="mt-1 text-[10px] text-studio-muted">留空则不修改已有密钥</p>
              </div>
            </>
          )}

          {error && <div className="studio-alert-error-compact">{error}</div>}
          {success && (
            <div className="rounded-lg border border-[rgb(var(--studio-diff-add-stat)/0.3)] bg-[rgb(var(--studio-diff-add-bg))] px-3 py-2 text-xs text-[rgb(var(--studio-diff-add-stat))]">
              {success}
            </div>
          )}
        </div>

        <div className="flex justify-end gap-2 border-t border-studio-border px-5 py-4">
          <button type="button" onClick={onClose} className="rounded-lg px-4 py-2 text-sm text-studio-muted">
            关闭
          </button>
          <button
            type="button"
            onClick={save}
            disabled={loading || saving}
            className="inline-flex items-center gap-1 rounded-lg bg-studio-accent px-4 py-2 text-sm text-studio-on-accent disabled:opacity-40"
          >
            {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
            保存
          </button>
        </div>
      </div>
    </div>
  );
}
