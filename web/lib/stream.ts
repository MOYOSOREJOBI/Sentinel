export type QueuePatch = { type: "upsert" | "delete"; incident: any };

function sub(path: string, eventName: string, onPatch: (p: any) => void, onError?: (err: any) => void) {
  const es = new EventSource(path, { withCredentials: true });
  const handle = (evt: MessageEvent) => {
    try { onPatch(JSON.parse((evt as MessageEvent).data)); } catch (err) { onError?.(err); }
  };
  es.onmessage = handle
  es.addEventListener(eventName, (evt) => handle(evt as MessageEvent))
  es.onerror = (err) => onError?.(err);
  return () => es.close();
}

export function subscribeQueuePatches(onPatch: (patch: QueuePatch) => void, onError?: (err: any) => void) {
  return sub("/api/sse/queue", "queue_patch", onPatch, onError);
}
export function subscribeCommandCenterPatches(onPatch: (patch: any) => void, onError?: (err: any) => void) {
  return sub("/api/sse/command-center", "command_center_patch", onPatch, onError);
}
export function subscribeTrustPatches(onPatch: (patch: any) => void, onError?: (err: any) => void) {
  return sub("/api/sse/trust", "trust_patch", onPatch, onError);
}
