// contexts/WebSocketContext.tsx
"use client";

import {
  createContext,
  useContext,
  useEffect,
  useRef,
  useState,
  ReactNode,
} from "react";

interface Notification {
  notification_id: number;
  user_id: number;
  type: string;
  message: string;
  read_status: boolean;
  created_at: string;
  related_user_id?: number;
  related_group_id?: number;
  sender_name?: string;
  group_name?: string;
}

interface WebSocketContextType {
  onlineUsers: number[];
  notifications: Notification[];
  isConnected: boolean;
  sendMessage: (data: any) => void;
  disconnect: () => void;
  addNotification: (notification: Notification) => void;
  removeNotification: (id: number) => void;
  clearNotifications: () => void;
}

const WebSocketContext = createContext<WebSocketContextType | undefined>(
  undefined
);

export function WebSocketProvider({ children }: { children: ReactNode }) {
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  const [onlineUsers, setOnlineUsers] = useState<number[]>([]);
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const [token, setToken] = useState<string | null>(null);

  // Load token from localStorage on mount
  useEffect(() => {
    setToken(localStorage.getItem("token"));
  }, []);

  const connect = () => {
    if (!token || wsRef.current?.readyState === WebSocket.OPEN) return;

    const wsUrl = process.env.NEXT_PUBLIC_WS_URL;
    if (!wsUrl) {
      console.error("NEXT_PUBLIC_WS_URL not configured");
      return;
    }

    try {
      const ws = new WebSocket(`${wsUrl}/ws`);
      wsRef.current = ws;

      ws.onopen = () => {
        console.log("WebSocket connected");
        setIsConnected(true);

        // Authenticate
        const authToken = token.startsWith("Bearer ")
          ? token.replace("Bearer ", "")
          : token;

        ws.send(JSON.stringify({ type: "auth", token: authToken }));
      };

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data);
          console.log("WebSocket message:", msg);

          switch (msg.type) {
            case "online_users":
              setOnlineUsers(
                Array.isArray(msg.online_users) ? msg.online_users : []
              );
              break;

            case "notification":
              if (msg.notification) {
                console.log("🔔 NOTIFICATION RECEIVED:", {
                  for_user_id: msg.notification.user_id,
                  my_user_id: localStorage.getItem("user_id"),
                  message: msg.notification.message,
                  type: msg.notification.type
                });

                // CRITICAL FIX: Only add notification if it's for THIS user
                const myUserId = parseInt(localStorage.getItem("user_id") || "0");
                if (msg.notification.user_id === myUserId) {
                  setNotifications((prev) => [msg.notification, ...prev]);

                  // Play notification sound
                  const audio = new Audio("/notification.mp3");
                  audio.volume = 0.5;
                  audio.play().catch(() => {});
                } else {
                  console.log("❌ Ignoring notification - not for this user");
                }
              }
              break;

            default:
              // Forward custom messages to listeners
              window.dispatchEvent(
                new CustomEvent("websocket-message", { detail: msg })
              );
              break;
          }
        } catch (err) {
          console.error("Error parsing WebSocket message:", err);
        }
      };

      ws.onclose = () => {
        console.log("WebSocket disconnected");
        setIsConnected(false);
        wsRef.current = null;

        // Attempt reconnect
        if (token) {
          reconnectTimeoutRef.current = setTimeout(connect, 3000);
        }
      };

      ws.onerror = (error) => {
        console.error("WebSocket error:", error);
        setIsConnected(false);
      };
    } catch (err) {
      console.error("Failed to create WebSocket connection:", err);
      setIsConnected(false);
    }
  };

  const disconnect = () => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }

    setIsConnected(false);
  };

  const sendMessage = (data: any) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(data));
    } else {
      console.warn("WebSocket not connected, cannot send message");
    }
  };

  const addNotification = (notification: Notification) => {
    setNotifications((prev) => [notification, ...prev]);
  };

  const removeNotification = (id: number) => {
    setNotifications((prev) =>
      prev.filter((n) => n.notification_id !== id)
    );
  };

  const clearNotifications = () => setNotifications([]);

  // Manage connection when token changes
  useEffect(() => {
    if (token) {
      connect();
    } else {
      disconnect();
    }

    return () => disconnect();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  // Listen for token changes across tabs or logout/login
  useEffect(() => {
    const handleStorageChange = (e: StorageEvent) => {
      if (e.key === "token") setToken(e.newValue);
    };

    window.addEventListener("storage", handleStorageChange);
    return () => window.removeEventListener("storage", handleStorageChange);
  }, []);

  return (
    <WebSocketContext.Provider
      value={{
        onlineUsers,
        notifications,
        isConnected,
        sendMessage,
        disconnect,
        addNotification,
        removeNotification,
        clearNotifications,
      }}
    >
      {children}
    </WebSocketContext.Provider>
  );
}

export function useWebSocket() {
  const context = useContext(WebSocketContext);
  if (!context) {
    throw new Error("useWebSocket must be used within WebSocketProvider");
  }
  return context;
}