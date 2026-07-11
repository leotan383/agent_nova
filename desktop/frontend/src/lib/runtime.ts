export type WriteEventPayload = {
  job_id?: string;
  chapter?: number;
  volume?: number;
  delta?: string;
  step?: string;
  message?: string;
  status?: string;
  error?: string;
  report?: string;
  content?: string;
  body?: string;
  setting_id?: string;
  phase?: string;
  turns?: string;
  batch_complete?: boolean;
  batch_index?: number;
  total_in_batch?: number;
  elapsed_ms?: number;
  chapters_processed?: number;
  skipped?: string[];
};

type WailsRuntime = {
  EventsOn: (eventName: string, callback: (...data: unknown[]) => void) => () => void;
  EventsOff: (eventName: string, ...additionalHandlerNames: string[]) => void;
};

function runtime(): WailsRuntime | undefined {
  return (window as unknown as { runtime?: WailsRuntime }).runtime;
}

export function eventsOn(eventName: string, handler: (payload: WriteEventPayload) => void): () => void {
  const rt = runtime();
  if (!rt) {
    return () => {};
  }
  return rt.EventsOn(eventName, (...args: unknown[]) => {
    const payload = (args[0] ?? {}) as WriteEventPayload;
    handler(payload);
  });
}

export const WRITE_EVENTS = {
  delta: "write:delta",
  step: "write:step",
  status: "write:status",
  done: "write:done",
  error: "write:error",
} as const;

export const REVISE_EVENTS = {
  delta: "revise:delta",
  status: "revise:status",
  done: "revise:done",
  error: "revise:error",
} as const;

export const COACH_EVENTS = {
  stream: "coach:stream",
  done: "coach:done",
  error: "coach:error",
} as const;

export const SELECTION_EVENTS = {
  delta: "selection:delta",
  status: "selection:status",
  done: "selection:done",
  error: "selection:error",
} as const;

export const DISCOVER_EVENTS = {
  stream: "discover:stream",
  done: "discover:done",
  error: "discover:error",
} as const;

export const INSPIRATION_EVENTS = {
  stream: "inspiration:stream",
  done: "inspiration:done",
  error: "inspiration:error",
} as const;

export const PLAN_EVENTS = {
  status: "plan:status",
  done: "plan:done",
  error: "plan:error",
} as const;

export const BOOK_READ_EVENTS = {
  status: "bookread:status",
  done: "bookread:done",
  error: "bookread:error",
} as const;

export const POLISH_EVENTS = {
  status: "polish:status",
  progress: "polish:progress",
  done: "polish:done",
  error: "polish:error",
} as const;

export const REVIEW_EVENTS = {
  status: "review:status",
  done: "review:done",
  error: "review:error",
} as const;

export const AI_DETECT_EVENTS = {
  status: "ai_detect:status",
  done: "ai_detect:done",
  error: "ai_detect:error",
} as const;

export const ENTITY_HISTORY_EVENTS = {
  status: "entity_history:status",
  done: "entity_history:done",
  error: "entity_history:error",
} as const;

export const SETTING_FILL_EVENTS = {
  status: "setting_fill:status",
  done: "setting_fill:done",
  error: "setting_fill:error",
} as const;
