"use client";

import { useEffect, useState } from "react";

interface Following {
  id: number;
  email: string;
  first_name: string;
  last_name: string;
  nickname: string;
  avatar: string;
  followed_at: string;
}

interface Props {
  apiBase: string;
  token: string;
  onClose: () => void;
}

export default function FollowingModal({ apiBase, token, onClose }: Props) {
  const [following, setFollowing] = useState<Following[]>([]);
  const [loading, setLoading] = useState(true);

  // Avatar URL normalizer
  const buildAvatarUrl = (avatar: string | null | undefined) => {
    if (!avatar) return "";
    let cleanPath = avatar.replace(/\\/g, "/");
    if (cleanPath.startsWith("http")) return cleanPath;
    if (!cleanPath.startsWith("/")) cleanPath = "/" + cleanPath;
    const base = apiBase.endsWith("/") ? apiBase.slice(0, -1) : apiBase;
    return `${base}${cleanPath}`;
  };

  useEffect(() => {
    const fetchFollowing = async () => {
      try {
        const res = await fetch(`${apiBase}/following`, {
          headers: { Authorization: token },
        });
        if (res.ok) {
          const data = await res.json();
          setFollowing(data);
        }
      } catch (err) {
        console.error("Error fetching following:", err);
      } finally {
        setLoading(false);
      }
    };
    fetchFollowing();
  }, [apiBase, token]);

  const getInitial = (f: Following) => {
    if (f.nickname) return f.nickname[0].toUpperCase();
    if (f.first_name) return f.first_name[0].toUpperCase();
    return f.email[0].toUpperCase();
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-40 backdrop-blur-sm flex justify-center items-center z-50">
      <div className="bg-gradient-to-br from-white to-gray-50 rounded-2xl shadow-2xl w-full max-w-md p-6 relative">
        {/* Title */}
        <h2 className="text-2xl font-extrabold text-gray-900 mb-5 border-b pb-3">
          Following
        </h2>

        {/* Content */}
        {loading ? (
          <p className="text-gray-600">Loading...</p>
        ) : following.length === 0 ? (
          <p className="text-gray-500 text-center py-6">
            You’re not following anyone yet.
          </p>
        ) : (
          <ul className="space-y-3 max-h-80 overflow-y-auto pr-2 custom-scroll">
            {following.map((f) => {
              const avatarUrl = buildAvatarUrl(f.avatar);
              return (
                <li
                  key={f.id}
                  className="flex items-center space-x-3 p-2 rounded-lg hover:bg-gray-100 transition"
                >
                  {avatarUrl ? (
                    <img
                      src={avatarUrl}
                      alt={f.nickname || f.email}
                      className="w-12 h-12 rounded-full object-cover border-2 border-indigo-500 shadow-sm"
                    />
                  ) : (
                    <div className="w-12 h-12 rounded-full bg-indigo-500 flex items-center justify-center text-white font-bold shadow-sm">
                      {getInitial(f)}
                    </div>
                  )}
                  <div>
                    <p className="font-semibold text-gray-900">
                      {f.nickname || f.email}
                    </p>
                    <p className="text-sm text-gray-600">
                      {f.first_name} {f.last_name}
                    </p>
                  </div>
                </li>
              );
            })}
          </ul>
        )}

        {/* Close button */}
        <button
          onClick={onClose}
          className="mt-6 w-full py-2.5 bg-gradient-to-r from-indigo-600 to-blue-600 text-white font-semibold rounded-xl shadow hover:from-indigo-700 hover:to-blue-700 transition"
        >
          Close
        </button>
      </div>
    </div>
  );
}
