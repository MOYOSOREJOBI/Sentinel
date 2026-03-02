"use client";
import { useEffect, useRef, useState } from "react";

type SSEStatus = "connecting" | "open" | "closed" | "error";
export function useSSE<T>(url: string, onMessage: (data: T) => void) {
  const [status, setStatus] = useState<SSEStatus>("connecting");
  const esRef = useRef<EventSource | null>(null);
  const onMessageRef = useRef(onMessage);
  useEffect(() => { onMessageRef.current = onMessage }, [onMessage])

  useEffect(() => {
    let closed = false;
    let retryMs = 2000;
    let flushTimer: ReturnType<typeof setTimeout> | null = null;
    let frame = 0;
    const queue: T[] = [];

    const flush = () => {
      flushTimer = null;
      frame = window.requestAnimationFrame(() => {
        const batch = queue.splice(0, queue.length);
        for (const item of batch) {
          onMessageRef.current(item);
        }
      });
    };

    const connect = () => {
      if (closed) return;
      setStatus("connecting");
      const es = new EventSource(url, { withCredentials: true });
      esRef.current = es;
      const handle = (evt: MessageEvent) => {
        try {
          queue.push(JSON.parse(evt.data));
          if (!flushTimer) {
            flushTimer = setTimeout(flush, 250);
          }
        } catch { }
      }

      es.onopen = () => { retryMs = 500; setStatus("open"); };
      es.onerror = () => {
        setStatus("error");
        es.close();
        if (closed) return;
        setTimeout(connect, retryMs);
        retryMs = Math.min(retryMs * 2, 15000);
      };
      es.onmessage = handle
      es.addEventListener("alert", (evt) => handle(evt as MessageEvent))
      es.addEventListener("queue_patch", (evt) => handle(evt as MessageEvent))
      es.addEventListener("command_center_patch", (evt) => handle(evt as MessageEvent))
      es.addEventListener("trust_patch", (evt) => handle(evt as MessageEvent))
    };

    connect();
    return () => {
      closed = true;
      if (flushTimer) clearTimeout(flushTimer);
      if (frame) window.cancelAnimationFrame(frame);
      esRef.current?.close();
      setStatus("closed");
    };
  }, [url]);

  return { status };
}
