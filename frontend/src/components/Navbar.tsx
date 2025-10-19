"use client";

import Link from "next/link";
import { useRouter } from "next/router";
import { useAuth } from "../context/AuthContext";
import { useWebSocket } from "../context/WebSocketContext";
import { useState, useEffect } from "react";
import NotificationsDropdown from "./NotificationsDropdown";

interface UserProfile {
  id: number;
  nickname: string;
  avatar: string;
}

const apiBase =
  typeof window !== "undefined"
    ? process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8088"
    : "";

export default function Navbar() {
  const { isSignedIn, logout } = useAuth();
  const router = useRouter();

  // Skip websocket on auth pages
  const isAuthPage = ["/signin", "/signup"].includes(router.pathname);
  const ws = !isAuthPage ? useWebSocket() : null;

  const [userProfile, setUserProfile] = useState<UserProfile | null>(null);
  const [token, setToken] = useState<string | null>(null);

  const buildAvatarUrl = (avatar?: string | null) => {
    if (!avatar) return "";
    const cleanPath = avatar.replace(/\\/g, "/");
    if (cleanPath.startsWith("http")) return cleanPath;

    const base = apiBase.endsWith("/") ? apiBase.slice(0, -1) : apiBase;
    const path = cleanPath.startsWith("/") ? cleanPath : `/${cleanPath}`;
    return `${base}${path}`;
  };

  const fetchCurrentUserInfo = async () => {
    const storedToken = localStorage.getItem("token");
    if (!storedToken) {
      setUserProfile(null);
      setToken(null);
      return;
    }

    setToken(storedToken);

    try {
      const res = await fetch(`${apiBase}/user/profile/details`, {
        headers: { Authorization: storedToken },
      });

      if (!res.ok) return;

      const data = await res.json();
      setUserProfile(data);

      localStorage.setItem("user_id", data.id.toString());
      localStorage.setItem("user_nickname", data.nickname);
      if (data.avatar) localStorage.setItem("user_avatar", data.avatar);
    } catch (err) {
      console.error("Failed to fetch current user info:", err);
    }
  };

  const handleLogout = () => {
    ws?.disconnect?.();
    logout();

    setUserProfile(null);
    setToken(null);

    ["token", "user_id", "user_nickname", "user_avatar"].forEach((key) =>
      localStorage.removeItem(key)
    );

    router.push("/signin");
  };

  useEffect(() => {
    fetchCurrentUserInfo();

    const handleRouteChange = () => fetchCurrentUserInfo();
    router.events.on("routeChangeComplete", handleRouteChange);

    return () => {
      router.events.off("routeChangeComplete", handleRouteChange);
    };
  }, [isSignedIn, router]);

  return (
    <nav className="bg-gray-800 text-white p-4 flex justify-between items-center">
      <Link href="/chats" className="font-bold text-lg">
        Talakee
      </Link>

      <div className="flex items-center space-x-4">
        {!isSignedIn ? (
          <>
            <Link href="/signup">Sign Up</Link>
            <Link href="/signin">Sign In</Link>
          </>
        ) : (
          <>
            {userProfile ? (
              <Link
                href="/profile"
                className="flex items-center space-x-2 hover:bg-gray-700 px-3 py-2 rounded transition-colors"
              >
                <div className="w-8 h-8 rounded-full bg-gray-600 overflow-hidden">
                  {userProfile.avatar ? (
                    <img
                      src={buildAvatarUrl(userProfile.avatar)}
                      alt="avatar"
                      className="w-full h-full object-cover"
                    />
                  ) : (
                    <div className="w-full h-full flex items-center justify-center text-sm bg-gray-500 font-medium">
                      {userProfile.nickname?.[0]?.toUpperCase() ?? "U"}
                    </div>
                  )}
                </div>
                <span className="text-sm font-medium">{userProfile.nickname}</span>
              </Link>
            ) : (
              <span className="text-sm">Loading...</span>
            )}

            {token && <NotificationsDropdown token={token} apiBase={apiBase} />}

            <button
              onClick={handleLogout}
              className="bg-red-500 px-3 py-2 rounded hover:bg-red-600 transition text-sm"
            >
              Sign Out
            </button>
          </>
        )}
      </div>
    </nav>
  );
}
