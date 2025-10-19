// components/NotificationsDropdown.tsx
import { useState, useEffect, useRef } from "react";
import { BellIcon } from "@heroicons/react/24/outline";
import { useRouter } from "next/router";
import { useWebSocket } from "../context/WebSocketContext";

interface NotificationsDropdownProps {
  apiBase: string;
  token: string;
}

export default function NotificationsDropdown({
  apiBase,
  token,
}: NotificationsDropdownProps) {
  const [dropdownOpen, setDropdownOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const router = useRouter();

  // Get notifications directly from WebSocket context
  const { notifications, removeNotification, clearNotifications } = useWebSocket();
  
  const unreadCount = notifications.filter(n => !n.read_status).length;

  const fetchNotifications = async () => {
    if (!token) return;

    setLoading(true);
    try {
      const res = await fetch(
        `${apiBase}/notifications?limit=20&offset=0&unread_only=true`,
        {
          method: "GET",
          headers: {
            Authorization: token.startsWith("Bearer ")
              ? token
              : `Bearer ${token}`,
            "Content-Type": "application/json",
          },
        }
      );

      if (!res.ok) {
        console.error("Failed to fetch notifications");
        return;
      }

      const data = await res.json();
      console.log("Fetched notifications:", data);
    } catch (err) {
      console.error("Error fetching notifications:", err);
    } finally {
      setLoading(false);
    }
  };

  const markAsRead = async (id: number) => {
    try {
      const res = await fetch(
        `${apiBase}/notifications/read?notification_id=${id}`,
        {
          method: "PUT",
          headers: {
            Authorization: token.startsWith("Bearer ")
              ? token
              : `Bearer ${token}`,
            "Content-Type": "application/json",
          },
        }
      );

      if (!res.ok) return;

      // Remove from context
      removeNotification(id);
    } catch (err) {
      console.error("Error marking notification read:", err);
    }
  };

  const markAllAsRead = async () => {
    try {
      const res = await fetch(`${apiBase}/notifications/read-all`, {
        method: "PUT",
        headers: {
          Authorization: token.startsWith("Bearer ")
            ? token
            : `Bearer ${token}`,
          "Content-Type": "application/json",
        },
      });
      if (!res.ok) return;

      clearNotifications();
    } catch (err) {
      console.error("Error marking all notifications read:", err);
    }
  };

  const handleDropdownToggle = () => {
    setDropdownOpen(!dropdownOpen);
    if (!dropdownOpen) {
      fetchNotifications();
    }
  };

  const handleGoToRequests = () => {
    setDropdownOpen(false);
    
    // If already on requests page, force reload
    if (router.pathname === "/requests") {
      router.reload();
    } else {
      router.push("/requests");
    }
  };

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node)
      ) {
        setDropdownOpen(false);
      }
    };

    if (dropdownOpen) {
      document.addEventListener("mousedown", handleClickOutside);
    }

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [dropdownOpen]);

  // Fetch initial notifications on mount
  useEffect(() => {
    if (token) {
      fetchNotifications();
    }
  }, [token]);

  // Debug: Log when notifications change
  useEffect(() => {
    console.log("Notifications updated in dropdown:", notifications);
  }, [notifications]);

  const getNotificationIcon = (type: string) => {
    switch (type) {
      case "follow_request":
        return "👤";
      case "group_invite":
        return "👥";
      case "group_request":
        return "📝";
      case "group_event":
        return "📅";
      case "group_accepted":
        return "✅";
      case "group_rejected":
        return "❌";
      default:
        return "🔔";
    }
  };

  return (
    <div className="relative" ref={dropdownRef}>
      {/* Bell button */}
      <button
        onClick={handleDropdownToggle}
        className="relative p-2 hover:bg-gray-700 rounded-full transition-colors"
        title="Notifications"
      >
        <BellIcon className="w-6 h-6 text-white" />
        {unreadCount > 0 && (
          <span className="absolute -top-1 -right-1 bg-red-500 text-white text-xs px-1.5 py-0.5 rounded-full min-w-[1.25rem] h-5 flex items-center justify-center">
            {unreadCount > 99 ? "99+" : unreadCount}
          </span>
        )}
      </button>

      {/* Dropdown */}
      {dropdownOpen && (
        <div className="absolute right-0 mt-2 w-80 bg-white shadow-lg rounded-lg z-50 border border-gray-200">
          {/* Header */}
          <div className="p-3 font-semibold border-b border-gray-200 flex justify-between items-center">
            <span className="text-gray-800">Notifications ({notifications.length})</span>
            {unreadCount > 0 && (
              <button
                onClick={markAllAsRead}
                className="text-xs text-blue-600 hover:text-blue-800 font-medium"
              >
                Mark all read
              </button>
            )}
          </div>

          {/* List */}
          <div className="max-h-80 overflow-y-auto">
            {loading && (
              <div className="p-4 text-center text-gray-500 text-sm">
                Loading notifications...
              </div>
            )}
            {!loading && notifications.length === 0 && (
              <div className="p-4 text-center text-gray-500 text-sm">
                No new notifications
              </div>
            )}
            {!loading &&
              notifications.map((n) => (
                <div
                  key={n.notification_id}
                  onClick={() => markAsRead(n.notification_id)}
                  className="p-3 border-b border-gray-100 hover:bg-gray-50 cursor-pointer transition-colors last:border-b-0"
                >
                  <div className="flex items-start space-x-3">
                    <span className="text-lg mt-0.5">
                      {getNotificationIcon(n.type)}
                    </span>
                    <div className="flex-1 min-w-0">
                      <div className="text-sm text-gray-800 font-medium mb-1">
                        {n.message}
                      </div>
                      <div className="text-xs text-gray-500">
                        {new Date(n.created_at).toLocaleString()}
                      </div>
                      {n.type && (
                        <div className="text-xs text-blue-600 mt-1 capitalize">
                          {n.type.replace(/_/g, " ")}
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              ))}
          </div>

          {/* Footer links */}
          <div className="p-2 border-t border-gray-200 space-y-2">
            <button
              onClick={handleGoToRequests}
              className="block w-full text-center text-sm text-green-600 hover:text-green-800 font-medium py-2 rounded-md bg-green-50 hover:bg-green-100"
            >
              Manage pending requests
            </button>
          </div>
        </div>
      )}
    </div>
  );
}