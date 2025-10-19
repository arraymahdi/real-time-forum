// pages/group/[id]/members.tsx
"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/router";
import { ChevronRight, ArrowLeft } from "lucide-react";
import { useWebSocket } from "../../../../context/WebSocketContext";

interface GroupMemberFromAPI {
  user_id: number;
  user_name: string;
  role?: string;
  status?: string;
  joined_at?: string;
  avatar?: string;
}

export default function GroupMembersPage() {
  const router = useRouter();
  const { id } = router.query;

  const [members, setMembers] = useState<GroupMemberFromAPI[]>([]);
  const [loading, setLoading] = useState(true);
  const [lastError, setLastError] = useState<string | null>(null);

  // Use centralized WebSocket for online users
  const { onlineUsers } = useWebSocket();

  const apiBase = process.env.NEXT_PUBLIC_API_BASE_URL;

  useEffect(() => {
    if (!id) return;
    const token = localStorage.getItem("token");

    if (!apiBase) {
      setLastError("API base URL not configured (NEXT_PUBLIC_API_BASE_URL).");
      setLoading(false);
      return;
    }

    // Fetch group members
    const fetchMembers = async () => {
      try {
        setLastError(null);
        const tryPaths = [
          `${apiBase}/chat/group/${id}/members`,
          `${apiBase}/group/${id}/members`,
        ];

        let res: Response | null = null;
        let data: any = null;
        let success = false;
        for (const p of tryPaths) {
          try {
            res = await fetch(p, {
              headers: token ? { Authorization: `Bearer ${token}` } : undefined,
            });
            if (res.ok) {
              data = await res.json();
              success = true;
              break;
            }
          } catch (err) {
            console.warn(`Fetch to ${p} failed:`, err);
          }
        }
        if (!success) throw new Error("All member endpoints failed");

        let items: any[] = [];
        if (Array.isArray(data)) items = data;
        else if (Array.isArray(data.members)) items = data.members;
        else if (Array.isArray(data.data)) items = data.data;

        // normalize basic members
        const normalized = items
          .map((it: any) => ({
            user_id: it.user_id ?? it.id ?? null,
            user_name: it.user_name ?? it.name ?? "",
            role: it.role,
            status: it.status,
            joined_at: it.joined_at,
          }))
          .filter((m: any) => m.user_id !== null);

        // fetch avatar for each member
        const withAvatars = await Promise.all(
          normalized.map(async (member) => {
            try {
              const res = await fetch(`${apiBase}/user/${member.user_id}`, {
                headers: token ? { Authorization: `Bearer ${token}` } : undefined,
              });
              if (res.ok) {
                const userData = await res.json();
                return { ...member, avatar: userData.avatar };
              }
            } catch (e) {
              console.warn(`Could not fetch avatar for user ${member.user_id}`);
            }
            return member;
          })
        );

        setMembers(withAvatars);
      } catch (err: any) {
        console.error("Failed to fetch members:", err);
        setMembers([]);
        setLastError(String(err?.message ?? err));
      } finally {
        setLoading(false);
      }
    };

    fetchMembers();
  }, [id, apiBase]);

  const isOnline = (userId?: number | null) =>
    userId ? onlineUsers.includes(userId) : false;

  const userChat = (user_id: number) => {
    router.push(`/chat/user/${user_id}`);
  };

  if (loading) {
    return <p className="text-center text-gray-500 mt-10">Loading members...</p>;
  }

  return (
    <div className="max-w-2xl mx-auto p-4">
      {/* Back button */}
      <button
        onClick={() => router.back()}
        className="flex items-center text-blue-600 mb-4 hover:underline"
      >
        <ArrowLeft className="w-5 h-5 mr-1" /> Back
      </button>

      <h1 className="text-2xl font-bold mb-4">Group Members</h1>

      {lastError && (
        <div className="mb-4 p-3 bg-yellow-50 border border-yellow-200 text-yellow-800 rounded">
          <strong>Warning:</strong> {lastError}
        </div>
      )}

      <ul className="divide-y divide-gray-200 bg-white shadow rounded-lg">
        {members.length === 0 ? (
          <li className="p-4 text-gray-500 text-center">No members found</li>
        ) : (
          members.map((member) => (
            <li
              key={member.user_id}
              className="p-4 flex justify-between items-center hover:bg-gray-50 cursor-pointer"
              onClick={() => userChat(member.user_id)}
            >
              <div className="flex items-center gap-3">
                {member.avatar ? (
                  <img
                    src={`${process.env.NEXT_PUBLIC_API_BASE_URL}/${member.avatar}`}
                    alt={member.user_name}
                    className="w-10 h-10 rounded-full object-cover"
                  />
                ) : (
                  <div className="w-10 h-10 rounded-full bg-blue-500 flex items-center justify-center text-white font-bold">
                    {member.user_name.charAt(0).toUpperCase()}
                  </div>
                )}

                <div>
                  <p className="font-semibold text-gray-900">{member.user_name}</p>
                  <div className="text-sm">
                    <span
                      className={`mr-2 ${
                        isOnline(member.user_id) ? "text-green-600" : "text-gray-400"
                      }`}
                    >
                      {isOnline(member.user_id) ? "Online" : "Offline"}
                    </span>
                    {member.role && (
                      <span className="text-xs text-blue-500">
                        • {member.role}
                      </span>
                    )}
                  </div>
                </div>
              </div>
              <ChevronRight className="w-5 h-5 text-gray-400" />
            </li>
          ))
        )}
      </ul>
    </div>
  );
}