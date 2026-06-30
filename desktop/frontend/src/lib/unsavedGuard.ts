export type UnsavedChoice = "save" | "discard" | "cancel";

export type UnsavedEditorHandle = {
  isDirty: () => boolean;
  save: () => Promise<boolean>;
  discard: () => void;
};

const editors = new Set<UnsavedEditorHandle>();

type PromptFn = (resolve: (choice: UnsavedChoice) => void) => void;
let promptHandler: PromptFn | null = null;

export function registerUnsavedEditor(handle: UnsavedEditorHandle): () => void {
  editors.add(handle);
  return () => editors.delete(handle);
}

export function hasUnsavedChanges(): boolean {
  for (const editor of editors) {
    if (editor.isDirty()) return true;
  }
  return false;
}

function getDirtyEditor(): UnsavedEditorHandle | null {
  for (const editor of editors) {
    if (editor.isDirty()) return editor;
  }
  return null;
}

export function setUnsavedPromptHandler(handler: PromptFn | null) {
  promptHandler = handler;
}

export async function confirmUnsavedLeave(): Promise<boolean> {
  const editor = getDirtyEditor();
  if (!editor) return true;

  const choice = await new Promise<UnsavedChoice>((resolve) => {
    if (promptHandler) {
      promptHandler(resolve);
      return;
    }
    resolve(window.confirm("有未保存的更改，确定离开吗？") ? "discard" : "cancel");
  });

  if (choice === "cancel") return false;
  if (choice === "discard") {
    editor.discard();
    return true;
  }
  const ok = await editor.save();
  return ok;
}
