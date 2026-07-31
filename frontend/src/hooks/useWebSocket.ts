import { useEffect, useRef, useState } from "react";
import type { IncomingPayload, WsPayload } from "../types";

export function useWebSocket(
  code: string,
  identity: string,
  onMessage: (p: WsPayload) => void,
) {
  const [connected, setConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);
  const handlerRef = useRef(onMessage);

  useEffect(() => {
    handlerRef.current = onMessage;
  });

  const sendRef = useRef<(data: IncomingPayload) => void>(() => {});
  sendRef.current = (data) => {
    const ws = wsRef.current;
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(data));
    }
  };

  useEffect(() => {
    let closed = false;
    let ws: WebSocket | null = null;
    let retry: number | undefined;

    const connect = () => {
      if (closed) return;
      const proto = location.protocol === "https:" ? "wss" : "ws";
      ws = new WebSocket(
        `${proto}://${location.host}/ws?code=${encodeURIComponent(code)}&user=${encodeURIComponent(identity)}`,
      );
      wsRef.current = ws;
      setConnected(false);
      ws.onopen = () => setConnected(true);
      ws.onmessage = (ev) => {
        try {
          handlerRef.current(JSON.parse(ev.data) as WsPayload);
        } catch {
          /* ignore malformed */
        }
      };
      ws.onclose = () => {
        setConnected(false);
        if (!closed) retry = window.setTimeout(connect, 2000);
      };
      ws.onerror = () => {
        try {
          ws?.close();
        } catch {
          /* noop */
        }
      };
    };

    connect();
    return () => {
      closed = true;
      if (retry) window.clearTimeout(retry);
      ws?.close();
      wsRef.current = null;
    };
  }, [code, identity]);

  return { connected, send: sendRef.current };
}
