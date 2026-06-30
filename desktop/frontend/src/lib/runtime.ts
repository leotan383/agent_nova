export type WriteEventPayload = {
  job_id?: string;
  chapter: number;
  delta?: string;
  step?: string;
  message?: string;
  status?: string;
  error?: string;
  report?: string;
  content?: string;
  phase?: string;
  turns?: string;
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
