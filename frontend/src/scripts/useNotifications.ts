import { useEffect, useRef } from 'react';
import { apiUrl } from './api.ts';

export type Notification = { type: string; data?: unknown };

const wsUrl = `${apiUrl.replace(/^http/, 'ws')}/ws`;
const RETRY_MS = 3000;

// Opens one websocket to the backend hub while `enabled`, reconnecting on
// drop. The httpOnly session cookie is sent with the handshake, so the
// backend's Authenticate can read it like any other request.
export function useNotifications(
  onMessage: (n: Notification) => void,
  enabled = true,
) {
  const handler = useRef(onMessage);
  handler.current = onMessage;

  useEffect(() => {
    if (!enabled) return;

    let ws: WebSocket | undefined;
    let retry: ReturnType<typeof setTimeout> | undefined;
    let stopped = false;

    const connect = () => {
      ws = new WebSocket(wsUrl);
      ws.onmessage = (e) => {
        try {
          handler.current(JSON.parse(e.data));
        } catch {
          // ignore malformed frames
        }
      };
      ws.onclose = () => {
        if (!stopped) retry = setTimeout(connect, RETRY_MS);
      };
    };

    connect();
    return () => {
      stopped = true;
      clearTimeout(retry);
      ws?.close();
    };
  }, [enabled]);
}
