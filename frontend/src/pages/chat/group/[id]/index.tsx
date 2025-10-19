"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/router";

interface GroupMessage {
  message_id?: number;
  group_id: number;
  sender_id: number;
  content: string;
  media?: string;
  created_at: string;
  sender_name: string;
}

interface GroupInfo {
  group_id: number;
  title: string;
  description?: string;
  member_count: number;
}

const EMOJI_CATEGORIES = {
  smileys: ["😀", "😃", "😄", "😁", "😆", "😅", "😂", "🤣", "😊", "😇", "🙂", "🙃", "😉", "😌", "😍", "🥰", "😘", "😗", "😙", "😚", "😋", "😛", "😝", "😜", "🤪", "🤨", "🧐", "🤓", "😎", "🤩", "🥳"],
  hearts: ["❤️", "🧡", "💛", "💚", "💙", "💜", "🖤", "🤍", "🤎", "💔", "❣️", "💕", "💞", "💓", "💗", "💖", "💘", "💝", "💟"],
  gestures: ["👍", "👎", "👌", "🤌", "🤏", "✌️", "🤞", "🤟", "🤘", "🤙", "👈", "👉", "👆", "🖕", "👇", "☝️", "👋", "🤚", "🖐", "✋", "🖖", "👏", "🙌", "🤝", "🙏"],
  activities: ["⚽", "🏀", "🏈", "⚾", "🥎", "🎾", "🏐", "🏉", "🥏", "🎱", "🪀", "🏓", "🏸", "🏒", "🏑", "🥍", "🏏", "🪃", "🥅", "⛳", "🪁", "🏹", "🎣", "🤿", "🥊", "🥋", "🎽"],
  objects: ["📱", "💻", "⌚", "📺", "📻", "🎮", "💡", "🔦", "🕯", "🪔", "📚", "📖", "📝", "✏️", "🖊", "🖋", "🖌", "🖍", "📎", "📍", "📌", "🔑", "🔒", "🔓"],
  nature: ["🌸", "🌺", "🌻", "🌷", "🌹", "🥀", "🌲", "🌳", "🌴", "🌵", "🌾", "🌿", "🍀", "🍃", "🍂", "🍁", "🌰", "🌱", "🌗", "🌝", "🌛", "🌜", "🌚", "🌕", "🌖", "🌔", "🌓", "🌒", "🌑"]
};

export default function GroupChatPage() {
  const router = useRouter();
  const { id } = router.query;

  const [messages, setMessages] = useState<GroupMessage[]>([]);
  const [input, setInput] = useState("");
  const [groupInfo, setGroupInfo] = useState<GroupInfo | null>(null);
  const [currentUserId, setCurrentUserId] = useState<number | null>(null);
  const [currentUserNickname, setCurrentUserNickname] = useState("");
  const [offset, setOffset] = useState(0);
  const [hasMore, setHasMore] = useState(true);
  const [showEmojiPicker, setShowEmojiPicker] = useState(false);
  const [activeEmojiCategory, setActiveEmojiCategory] = useState<keyof typeof EMOJI_CATEGORIES>("smileys");

  const ws = useRef<WebSocket | null>(null);
  const messagesEndRef = useRef<HTMLDivElement | null>(null);
  const containerRef = useRef<HTMLDivElement | null>(null);
  const inputRef = useRef<HTMLInputElement | null>(null);
  const emojiPickerRef = useRef<HTMLDivElement | null>(null);

  const token =
    typeof window !== "undefined" ? localStorage.getItem("token") : null;

  // Close emoji picker when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (emojiPickerRef.current && !emojiPickerRef.current.contains(event.target as Node)) {
        setShowEmojiPicker(false);
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  // ✅ Get current user info
  const fetchCurrentUserInfo = async () => {
    if (!token) return;
    try {
      const storedUserId = localStorage.getItem("user_id");
      const storedUserNickname = localStorage.getItem("user_nickname");

      if (storedUserId && storedUserNickname) {
        setCurrentUserId(parseInt(storedUserId));
        setCurrentUserNickname(storedUserNickname);
        return;
      }

      const res = await fetch(
        `${process.env.NEXT_PUBLIC_API_BASE_URL}/user/profile`,
        { headers: { Authorization: `Bearer ${token}` } }
      );

      if (res.ok) {
        const userData = await res.json();
        setCurrentUserId(userData.id);
        setCurrentUserNickname(userData.nickname);
        localStorage.setItem("user_id", userData.id.toString());
        localStorage.setItem("user_nickname", userData.nickname);
      }
    } catch (err) {
      console.error("Failed to fetch current user info:", err);
    }
  };

  // ✅ Get group info
  const fetchGroupInfo = async () => {
    if (!id) return;
    try {
      const raw = localStorage.getItem("chat-list");
      if (raw) {
        const chats = JSON.parse(raw);
        const found = chats.find(
          (c: any) => String(c.id) === String(id) && c.type === "group"
        );
        if (found) {
          setGroupInfo({
            group_id: found.id,
            title: found.name,
            member_count: found.member_count || 0,
          });
          return;
        }
      }

      const res = await fetch(
        `${process.env.NEXT_PUBLIC_API_BASE_URL}/group/${id}`,
        { headers: { Authorization: `Bearer ${token}` } }
      );

      if (res.ok) {
        const groupData = await res.json();
        setGroupInfo(groupData);
      } else {
        setGroupInfo({
          group_id: Number(id),
          title: `Group ${id}`,
          member_count: 0,
        });
      }
    } catch (err) {
      console.error("Failed to fetch group info:", err);
      setGroupInfo({
        group_id: Number(id),
        title: `Group ${id}`,
        member_count: 0,
      });
    }
  };

  // ✅ Fetch messages
  const fetchMessages = async (newOffset = 0) => {
    if (!id || !token) return;
    try {
      const res = await fetch(
        `${process.env.NEXT_PUBLIC_API_BASE_URL}/group-messages?group_id=${id}&offset=${newOffset}`,
        { headers: { Authorization: `Bearer ${token}` } }
      );
      if (!res.ok) throw new Error("Failed to fetch messages");
      const data = await res.json();

      if (data.length === 0) {
        setHasMore(false);
        return;
      }

      if (newOffset === 0) {
        setMessages(data.reverse());
        scrollToBottom();
      } else {
        setMessages((prev) => [...data.reverse(), ...prev]);
      }
    } catch (err) {
      console.error(err);
    }
  };

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  const isMyMessage = (message: GroupMessage) => {
    return currentUserId !== null && message.sender_id === currentUserId;
  };

  // ✅ Add emoji to input
  const addEmoji = (emoji: string) => {
    const cursorPosition = inputRef.current?.selectionStart || input.length;
    const textBefore = input.substring(0, cursorPosition);
    const textAfter = input.substring(cursorPosition);
    const newText = textBefore + emoji + textAfter;
    
    setInput(newText);
    setShowEmojiPicker(false);
    
    // Focus back on input and set cursor position
    setTimeout(() => {
      if (inputRef.current) {
        inputRef.current.focus();
        inputRef.current.setSelectionRange(cursorPosition + emoji.length, cursorPosition + emoji.length);
      }
    }, 10);
  };

  // ✅ Setup
  useEffect(() => {
    if (!id || !token) return;

    fetchCurrentUserInfo();
    fetchGroupInfo();
    fetchMessages(0);

    const socket = new WebSocket(`${process.env.NEXT_PUBLIC_WS_URL}/ws`);
    ws.current = socket;

    socket.onopen = () => {
      socket.send(JSON.stringify({ token }));
    };

    socket.onmessage = (event) => {
      const msg = JSON.parse(event.data);
      if (msg.type === "group" && msg.group_id === Number(id)) {
        const groupMsg: GroupMessage = {
          message_id: msg.id || Date.now(),
          group_id: msg.group_id,
          sender_id: msg.sender_id,
          content: msg.content,
          media: msg.media,
          created_at: msg.sent_at,
          sender_name: msg.sender_name,
        };
        setMessages((prev) => [...prev, groupMsg]);
        scrollToBottom();
      }
    };

    return () => {
      socket.close();
    };
  }, [id, token]);

  // ✅ Infinite scroll
  useEffect(() => {
    const div = containerRef.current;
    if (!div) return;

    const handleScroll = () => {
      if (div.scrollTop === 0 && hasMore) {
        const newOffset = offset + 10;
        setOffset(newOffset);
        fetchMessages(newOffset);
      }
    };

    div.addEventListener("scroll", handleScroll);
    return () => div.removeEventListener("scroll", handleScroll);
  }, [offset, hasMore, id, token]);

  // ✅ Send message
  const sendMessage = () => {
    if (!input.trim() || !ws.current || !currentUserId) return;
    if (ws.current.readyState !== WebSocket.OPEN) {
      console.warn("WebSocket not open yet");
      return;
    }

    const msg = {
      type: "group",
      group_id: Number(id),
      content: input.trim(),
    };

    ws.current.send(JSON.stringify(msg));

    setMessages((prev) => [
      ...prev,
      {
        message_id: Date.now(),
        group_id: Number(id),
        sender_id: currentUserId,
        content: input.trim(),
        created_at: new Date().toISOString(),
        sender_name: currentUserNickname,
      },
    ]);

    setInput("");
    scrollToBottom();
  };

  const groupMembers = () => {
    router.push(`/chat/group/${id}/members`);
  };

  return (
    <div className="flex flex-col h-screen">
      {/* Header with Back Arrow */}
      <div className="p-4 bg-gray-800 text-white flex items-center gap-3">
        <button onClick={() => router.back()}>
          <svg
            xmlns="http://www.w3.org/2000/svg"
            className="h-6 w-6"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            strokeWidth={2}
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M15 19l-7-7 7-7"
            />
          </svg>
        </button>
        <div>
          <div
            className="font-semibold text-lg cursor-pointer"
            onClick={() => groupMembers()}
          >
            {groupInfo?.title || `Group ${id}`}
          </div>
          {groupInfo?.member_count && (
            <div className="text-sm text-gray-300">
              {groupInfo.member_count} members
            </div>
          )}
        </div>
      </div>

      {/* Messages */}
      <div
        ref={containerRef}
        className="flex-1 overflow-y-auto p-4 bg-white text-gray-900"
      >
        {messages.map((m, i) => {
          const isMe = isMyMessage(m);
          return (
            <div key={i} className={`mb-3 ${isMe ? "text-right" : "text-left"}`}>
              <div className="text-xs text-gray-600">
                {isMe ? "Me" : m.sender_name}
              </div>
              <div
                className={`inline-block px-3 py-2 rounded-lg shadow break-words max-w-xs lg:max-w-md ${
                  isMe
                    ? "bg-blue-500 text-white rounded-br-none"
                    : "bg-gray-200 text-gray-900 rounded-bl-none"
                }`}
                style={{ fontSize: "16px", lineHeight: "1.4" }}
              >
                {m.content}
              </div>
              <div className="text-xs text-gray-500">
                {new Date(m.created_at).toLocaleTimeString()}
              </div>
            </div>
          );
        })}
        <div ref={messagesEndRef}></div>
      </div>

      {/* Emoji Picker */}
      {showEmojiPicker && (
        <div ref={emojiPickerRef} className="bg-white border-t border-gray-300 shadow-lg">
          {/* Category Tabs */}
          <div className="flex border-b border-gray-200 bg-gray-50">
            {Object.keys(EMOJI_CATEGORIES).map((category) => (
              <button
                key={category}
                onClick={() => setActiveEmojiCategory(category as keyof typeof EMOJI_CATEGORIES)}
                className={`px-4 py-2 text-sm font-medium capitalize ${
                  activeEmojiCategory === category
                    ? "text-blue-600 border-b-2 border-blue-600"
                    : "text-gray-600 hover:text-gray-800"
                }`}
              >
                {category}
              </button>
            ))}
          </div>
          
          {/* Emojis Grid */}
          <div className="p-4 h-48 overflow-y-auto">
            <div className="grid grid-cols-8 gap-2">
              {EMOJI_CATEGORIES[activeEmojiCategory].map((emoji, index) => (
                <button
                  key={index}
                  onClick={() => addEmoji(emoji)}
                  className="text-2xl p-1 rounded hover:bg-gray-100 transition-colors"
                  title={emoji}
                >
                  {emoji}
                </button>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Input */}
      <div className="p-4 bg-gray-100 flex items-end gap-2">
        {/* Emoji Button */}
        <button
          onClick={() => setShowEmojiPicker(!showEmojiPicker)}
          className={`p-2 rounded-full transition-colors ${
            showEmojiPicker ? "bg-blue-500 text-white" : "bg-gray-200 text-gray-600 hover:bg-gray-300"
          }`}
          title="Add emoji"
        >
          <span className="text-xl">😊</span>
        </button>

        {/* Input Field */}
        <input
          ref={inputRef}
          className="flex-1 border rounded-lg px-3 py-2 text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              sendMessage();
            }
          }}
          placeholder="Type a message..."
          style={{ fontSize: "16px" }}
        />

        {/* Send Button */}
        <button
          onClick={sendMessage}
          disabled={!input.trim()}
          className={`px-4 py-2 rounded-lg font-medium transition-colors ${
            input.trim()
              ? "bg-blue-600 text-white hover:bg-blue-700"
              : "bg-gray-300 text-gray-500 cursor-not-allowed"
          }`}
        >
          Send
        </button>
      </div>
    </div>
  );
}