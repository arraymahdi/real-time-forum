// hooks/useWebSocket.ts
import { useEffect, useRef, useState } from "react";

export function useWebSocket(token?: string) {
  const wsRef = useRef<WebSocket | null>(null);
  const [onlineUsers, setOnlineUsers] = useState<number[]>([]);
  const [notifications, setNotifications] = useState<any[]>([]);

  useEffect(() => {
    if (!token) return;
    const ws = new WebSocket(`${process.env.NEXT_PUBLIC_WS_URL}/ws`);
    wsRef.current = ws;

    ws.onopen = () => {
      ws.send(JSON.stringify({ type: "auth", token }));
    };

    ws.onmessage = (e) => {
      const msg = JSON.parse(e.data);
      if (msg.type === "online_users") setOnlineUsers(msg.online_users || []);
      if (msg.type === "notification") setNotifications((p) => [msg.notification, ...p]);
      // handle messages too if needed
    };

    ws.onclose = () => {
      wsRef.current = null;
    };

    return () => ws.close();
  }, [token]);

  const disconnect = () => {
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
  };

  return { onlineUsers, notifications, disconnect };
}
