// pages/chats.tsx
"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/router";
import { ChevronRight } from "lucide-react";
import { useWebSocket } from "../../context/WebSocketContext";

interface ChatItem {
  id: number;
  name: string;
  type: "user" | "group";
  avatar?: string;
  last_message_time?: string;
  last_message?: string;
  member_count?: number;
}

export default function ChatListPage() {
  const [chats, setChats] = useState<ChatItem[]>([]);
  const [loading, setLoading] = useState(true);
  const router = useRouter();
  const { onlineUsers } = useWebSocket();

  useEffect(() => {
    const token = localStorage.getItem("token");
    const apiBase = process.env.NEXT_PUBLIC_API_BASE_URL;

    if (!apiBase) {
      console.error("NEXT_PUBLIC_API_BASE_URL not set");
      return;
    }

    const fetchChats = async () => {
      try {
        const res = await fetch(`${apiBase}/chat-list`, {
          headers: token ? { Authorization: `Bearer ${token}` } : undefined,
        });
        if (!res.ok) throw new Error("Failed to fetch chat list");
        const data = await res.json();

        let items: ChatItem[] = data.chat_items || [];

        // Fetch avatars for users
        const updatedItems = await Promise.all(
          items.map(async (chat) => {
            if (chat.type === "user") {
              try {
                const userRes = await fetch(`${apiBase}/user/${chat.id}`, {
                  headers: token ? { Authorization: `Bearer ${token}` } : undefined,
                });
                if (userRes.ok) {
                  const userData = await userRes.json();
                  return { ...chat, avatar: userData.avatar };
                }
              } catch (err) {
                console.error(`Failed to fetch user ${chat.id}:`, err);
              }
            }
            return chat;
          })
        );

        setChats(updatedItems);
        localStorage.setItem("chat-list", JSON.stringify(updatedItems));
      } catch (err) {
        console.error("Chat fetch error:", err);
        setChats([]);
      } finally {
        setLoading(false);
      }
    };

    fetchChats();
  }, []);

  const goToChat = (chat: ChatItem) => {
    router.push(`/chat/${chat.type}/${chat.id}`);
  };

  if (loading) {
    return (
      <p className="text-center text-gray-500 mt-10 animate-pulse">
        Loading chats...
      </p>
    );
  }

  return (
    <div className="max-w-2xl mx-auto h-screen flex flex-col">
      {/* Header */}
      <div className="bg-gradient-to-r from-blue-600 to-indigo-600 text-white p-4 shadow-md rounded-b-xl">
        <h1 className="text-2xl font-bold">Chats</h1>
        <p className="text-sm text-blue-100">Stay connected with your friends & groups</p>
      </div>

      {/* Chat List */}
      <div className="flex-1 overflow-y-auto bg-gray-50 p-4">
        {chats.length === 0 ? (
          <div className="text-center text-gray-500 mt-20">
            No chats found 😢
          </div>
        ) : (
          <ul className="space-y-3">
            {chats.map((chat) => (
              <li
                key={`${chat.type}-${chat.id}`}
                onClick={() => goToChat(chat)}
                className="bg-white rounded-xl p-4 flex justify-between items-center shadow-sm hover:shadow-md hover:bg-gray-50 transition cursor-pointer"
              >
                {/* Left Section */}
                <div className="flex items-center gap-3">
                  {/* Avatar */}
                  <div className="relative">
                    {chat.avatar ? (
                      <img
                        src={`${process.env.NEXT_PUBLIC_API_BASE_URL}/${chat.avatar}`}
                        alt={chat.name}
                        className="w-12 h-12 rounded-full object-cover"
                      />
                    ) : (
                      <div className="w-12 h-12 rounded-full bg-gradient-to-br from-blue-500 to-indigo-500 flex items-center justify-center text-white font-bold text-lg">
                        {chat.name.charAt(0).toUpperCase()}
                      </div>
                    )}

                    {/* Status Dot */}
                    {chat.type === "user" && (
                      <span
                        className={`absolute bottom-0 right-0 w-3 h-3 rounded-full border-2 border-white ${
                          onlineUsers.includes(chat.id)
                            ? "bg-green-500"
                            : "bg-gray-400"
                        }`}
                      ></span>
                    )}
                  </div>

                  {/* Chat Info */}
                  <div>
                    <p className="font-semibold text-gray-900">{chat.name}</p>
                    {chat.type === "user" ? (
                      <span
                        className={`text-sm ${
                          onlineUsers.includes(chat.id)
                            ? "text-green-500"
                            : "text-gray-400"
                        }`}
                      >
                        {onlineUsers.includes(chat.id) ? "Online" : "Offline"}
                      </span>
                    ) : (
                      <span className="text-sm text-blue-500">
                        Group • {chat.member_count || 0} members
                      </span>
                    )}

                    {/* Last message */}
                    {chat.last_message && (
                      <p className="text-sm text-gray-500 truncate max-w-xs">
                        {chat.last_message}
                      </p>
                    )}
                  </div>
                </div>

                {/* Right Section */}
                <div className="flex flex-col items-end gap-1">
                  {chat.last_message_time && (
                    <span className="text-xs text-gray-400">
                      {new Date(chat.last_message_time).toLocaleTimeString([], {
                        hour: "2-digit",
                        minute: "2-digit",
                      })}
                    </span>
                  )}
                  <ChevronRight className="w-5 h-5 text-gray-400" />
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
