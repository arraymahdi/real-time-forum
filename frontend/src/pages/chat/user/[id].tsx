"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/router";
import { useWebSocket } from "../../../context/WebSocketContext";

interface Message {
  id?: number;
  sender_id: number;
  receiver_id: number;
  content: string;
  sent_at: string;
  type: string;
  sender_name: string;
}

const EMOJI_CATEGORIES = {
  smileys: ["😀", "😃", "😄", "😁", "😆", "😅", "😂", "🤣", "😊", "😇", "🙂", "🙃", "😉", "😌", "😍", "🥰", "😘", "😗", "😙", "😚", "😋", "😛", "😝", "😜", "🤪", "🤨", "🧐", "🤓", "😎", "🤩", "🥳"],
  hearts: ["❤️", "🧡", "💛", "💚", "💙", "💜", "🖤", "🤍", "🤎", "💔", "❣️", "💕", "💞", "💓", "💗", "💖", "💘", "💝", "💟"],
  gestures: ["👍", "👎", "👌", "🤌", "🤏", "✌️", "🤞", "🤟", "🤘", "🤙", "👈", "👉", "👆", "🖕", "👇", "☝️", "👋", "🤚", "🖐", "✋", "🖖", "👏", "🙌", "🤝", "🙏"],
  activities: ["⚽", "🏀", "🏈", "⚾", "🥎", "🎾", "🏐", "🏉", "🥏", "🎱", "🪀", "🏓", "🏸", "🏒", "🏑", "🥍", "🏏", "🪃", "🥅", "⛳", "🪁", "🏹", "🎣", "🤿", "🥊", "🥋", "🎽"],
  objects: ["📱", "💻", "⌚", "📺", "📻", "🎮", "💡", "🔦", "🕯", "🪔", "📚", "📖", "📝", "✏️", "🖊", "🖋", "🖌", "🖍", "📎", "📍", "📌", "🔑", "🔒", "🔓"],
  nature: ["🌸", "🌺", "🌻", "🌷", "🌹", "🥀", "🌲", "🌳", "🌴", "🌵", "🌾", "🌿", "🍀", "🍃", "🍂", "🍁", "🌰", "🌱", "🌗", "🌝", "🌛", "🌜", "🌚", "🌕", "🌖", "🌔", "🌓", "🌒", "🌑"]
};

export default function UserChatPage() {
  const router = useRouter();
  const { id } = router.query;

  const [token, setToken] = useState<string | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState("");
  const [nickname, setNickname] = useState("");
  const [avatar, setAvatar] = useState("");
  const [currentUserId, setCurrentUserId] = useState<number | null>(null);
  const [currentUserNickname, setCurrentUserNickname] = useState("");
  const [offset, setOffset] = useState(0);
  const [hasMore, setHasMore] = useState(true);
  const [typing, setTyping] = useState(false);
  const [showEmojiPicker, setShowEmojiPicker] = useState(false);
  const [activeEmojiCategory, setActiveEmojiCategory] = useState<
    keyof typeof EMOJI_CATEGORIES
  >("smileys");

  const typingTimeout = useRef<NodeJS.Timeout | null>(null);
  const messagesEndRef = useRef<HTMLDivElement | null>(null);
  const containerRef = useRef<HTMLDivElement | null>(null);
  const inputRef = useRef<HTMLInputElement | null>(null);
  const emojiPickerRef = useRef<HTMLDivElement | null>(null);

  const { onlineUsers, sendMessage: wsSendMessage } = useWebSocket();

  // ✅ Safely get localStorage data after client hydration
  useEffect(() => {
    if (typeof window !== "undefined") {
      const storedToken = localStorage.getItem("token");
      setToken(storedToken);
    }
  }, []);

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

  // Listen for WebSocket messages
  useEffect(() => {
    const handleWebSocketMessage = (event: CustomEvent) => {
      const msg = event.detail;
      
      if (msg.type === "private" && 
          (msg.sender_id === Number(id) || msg.receiver_id === Number(id))) {
        setMessages((prev) => [...prev, msg]);
        scrollToBottom();
      }

      if (msg.type === "typing" && msg.sender_id === Number(id)) {
        setTyping(true);
        if (typingTimeout.current) clearTimeout(typingTimeout.current);
        typingTimeout.current = setTimeout(() => setTyping(false), 2000);
      }
    };

    window.addEventListener('websocket-message', handleWebSocketMessage as EventListener);
    return () => window.removeEventListener('websocket-message', handleWebSocketMessage as EventListener);
  }, [id]);

  const normalizeAvatar = (path: string) => {
    if (!path) return "";
    if (path.startsWith("http")) return path;
    return `${process.env.NEXT_PUBLIC_API_BASE_URL}${path.startsWith("/") ? "" : "/"}${path}`;
  };

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

  const fetchNickname = async () => {
    if (!id) return;
    try {
      const raw = localStorage.getItem("chat-list");
      if (raw) {
        const chats = JSON.parse(raw);
        const found = chats.find((c: any) => String(c.id) === String(id));
        if (found && found.name) {
          setNickname(found.name);
          setAvatar(normalizeAvatar(found.avatar || ""));
          return;
        }
      }

      const res = await fetch(
        `${process.env.NEXT_PUBLIC_API_BASE_URL}/user/${id}`,
        { headers: { Authorization: `Bearer ${token}` } }
      );

      if (res.ok) {
        const userData = await res.json();
        setNickname(userData.nickname || `User ${id}`);
        setAvatar(normalizeAvatar(userData.avatar || ""));
      } else {
        setNickname(`User ${id}`);
      }
    } catch (err) {
      console.error("Failed to fetch user nickname:", err);
      setNickname(`User ${id}`);
    }
  };

  const fetchMessages = async (newOffset = 0) => {
    if (!id || !token) return;
    try {
      const res = await fetch(
        `${process.env.NEXT_PUBLIC_API_BASE_URL}/private-messages?other_user=${id}&offset=${newOffset}`,
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

  const isMyMessage = (message: Message) => {
    return currentUserId !== null && message.sender_id === currentUserId;
  };

  const addEmoji = (emoji: string) => {
    const cursorPosition = inputRef.current?.selectionStart || input.length;
    const textBefore = input.substring(0, cursorPosition);
    const textAfter = input.substring(cursorPosition);
    const newText = textBefore + emoji + textAfter;
    
    setInput(newText);
    setShowEmojiPicker(false);
    
    setTimeout(() => {
      if (inputRef.current) {
        inputRef.current.focus();
        inputRef.current.setSelectionRange(cursorPosition + emoji.length, cursorPosition + emoji.length);
      }
    }, 10);
  };

  useEffect(() => {
    if (!id || !token) return;

    fetchCurrentUserInfo();
    fetchNickname();
    fetchMessages(0);
  }, [id, token]);

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

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setInput(e.target.value);
    wsSendMessage({ type: "typing", receiver_id: Number(id) });
  };

  const sendMessage = () => {
    if (!input.trim() || !currentUserId) return;

    const msg = {
      type: "private",
      receiver_id: Number(id),
      content: input.trim(),
    };

    wsSendMessage(msg);

    setMessages((prev) => [
      ...prev,
      {
        id: Date.now(),
        sender_id: currentUserId,
        receiver_id: Number(id),
        content: input.trim(),
        sent_at: new Date().toISOString(),
        type: "private",
        sender_name: currentUserNickname,
      },
    ]);

    setInput("");
    scrollToBottom();
  };

  return (
    <div className="flex flex-col h-screen">
      <div className="p-4 bg-gray-800 text-white flex items-center gap-3">
        <button onClick={() => router.push("/chats")} className="mr-2">
          <svg xmlns="http://www.w3.org/2000/svg" className="h-6 w-6 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
        </button>

        <div className="w-10 h-10 rounded-full bg-gray-600 overflow-hidden">
          {avatar ? (
            <img src={avatar} alt="avatar" className="w-full h-full object-cover" />
          ) : (
            <div className="w-full h-full flex items-center justify-center text-lg bg-gray-500">
              {nickname ? nickname[0].toUpperCase() : "U"}
            </div>
          )}
        </div>

        <div>
          <div className="font-semibold">{nickname || `User ${id}`}</div>
          <div className="text-sm">
            {onlineUsers.includes(Number(id)) ? (
              <span className="text-green-400">Online</span>
            ) : (
              <span className="text-gray-400">Offline</span>
            )}
          </div>
        </div>
      </div>

      <div ref={containerRef} className="flex-1 overflow-y-auto p-4 bg-white text-gray-900">
        {messages.map((m, i) => {
          const isMe = isMyMessage(m);
          return (
            <div key={i} className={`mb-3 ${isMe ? "text-right" : "text-left"}`}>
              <div className="text-xs text-gray-600">{isMe ? "Me" : m.sender_name}</div>
              <div
                className={`inline-block px-3 py-2 rounded-lg shadow max-w-xs lg:max-w-md break-words ${
                  isMe ? "bg-blue-500 text-white rounded-br-none" : "bg-gray-200 text-gray-900 rounded-bl-none"
                }`}
                style={{ fontSize: "16px", lineHeight: "1.4" }}
              >
                {m.content}
              </div>
              <div className="text-xs text-gray-500">{new Date(m.sent_at).toLocaleTimeString()}</div>
            </div>
          );
        })}

        {typing && (
          <div className="mb-3 text-left">
            <div className="inline-block px-3 py-2 rounded-lg bg-gray-200 text-gray-600 text-sm animate-pulse">
              {nickname || "User"} is typing<span className="dot-animate ml-1">...</span>
            </div>
          </div>
        )}

        <div ref={messagesEndRef}></div>
      </div>

      {showEmojiPicker && (
        <div ref={emojiPickerRef} className="bg-white border-t border-gray-300 shadow-lg">
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

      <div className="p-4 bg-gray-100 flex items-end gap-2">
        <button
          onClick={() => setShowEmojiPicker(!showEmojiPicker)}
          className={`p-2 rounded-full transition-colors ${
            showEmojiPicker ? "bg-blue-500 text-white" : "bg-gray-200 text-gray-600 hover:bg-gray-300"
          }`}
          title="Add emoji"
        >
          <span className="text-xl">😊</span>
        </button>

        <input
          ref={inputRef}
          className="flex-1 border rounded-lg px-3 py-2 text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500"
          value={input}
          onChange={handleInputChange}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              sendMessage();
            }
          }}
          placeholder="Type a message..."
          style={{ fontSize: "16px" }}
        />

        <button
          onClick={sendMessage}
          disabled={!input.trim()}
          className={`px-4 py-2 rounded-lg font-medium transition-colors ${
            input.trim() ? "bg-blue-600 text-white hover:bg-blue-700" : "bg-gray-300 text-gray-500 cursor-not-allowed"
          }`}
        >
          Send
        </button>
      </div>

      <style jsx>{`
        .dot-animate::after {
          content: "";
          display: inline-block;
          width: 1em;
          text-align: left;
          animation: dots 1.5s steps(3, end) infinite;
        }
        @keyframes dots {
          0%, 20% { content: ""; }
          40% { content: "."; }
          60% { content: ".."; }
          80%, 100% { content: "..."; }
        }
      `}</style>
    </div>
  );
}
