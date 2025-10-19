"use client";

import { useState, useEffect } from "react";
import UserHeader from "@/components/UserHeader";

interface UserProfile {
  id: number;
  nickname: string;
  avatar?: string;
  profile_type: string;
  first_name?: string;
  last_name?: string;
}

const apiBase =
  typeof window !== "undefined"
    ? process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8088"
    : "";

export default function SearchPage() {
  const [users, setUsers] = useState<UserProfile[]>([]);
  const [searchTerm, setSearchTerm] = useState("");
  const [filteredUsers, setFilteredUsers] = useState<UserProfile[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const token = typeof window !== "undefined" ? localStorage.getItem("token") : null;

  useEffect(() => {
    if (!token) {
      setError("You must be signed in to search users.");
      setLoading(false);
      return;
    }

    const fetchUsers = async () => {
      try {
        const res = await fetch(`${apiBase}/users`, {
          headers: { Authorization: token || "" },
        });
        if (res.ok) {
          const data: UserProfile[] = await res.json();
          setUsers(data);
          setFilteredUsers(data);
        } else {
          setError("Failed to fetch users.");
        }
      } catch (err) {
        console.error(err);
        setError("Failed to fetch users.");
      } finally {
        setLoading(false);
      }
    };

    fetchUsers();
  }, [token]);

  useEffect(() => {
    const term = searchTerm.toLowerCase();
    setFilteredUsers(
      users.filter(
        (u) =>
          u.nickname.toLowerCase().includes(term) ||
          u.first_name?.toLowerCase().includes(term) ||
          u.last_name?.toLowerCase().includes(term)
      )
    );
  }, [searchTerm, users]);

  if (loading) return <p className="p-6">Loading users...</p>;
  if (error) return <p className="p-6 text-red-600">{error}</p>;

  return (
    <div className="min-h-screen bg-gray-50 py-10">
      <div className="max-w-2xl mx-auto bg-white rounded-xl shadow-lg p-6">
        <h1 className="text-2xl font-bold mb-4 text-gray-900">Search Users</h1>
        <input
          type="text"
          placeholder="Search by nickname or name..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          className="w-full p-3 mb-6 border rounded-lg focus:ring-2 focus:ring-blue-300 focus:outline-none text-gray-800"
        />

        {filteredUsers.length === 0 ? (
          <p className="text-gray-500">No users found.</p>
        ) : (
          <div className="space-y-4">
            {filteredUsers.map((user) => (
              <UserHeader
                key={user.id}
                userId={user.id}
                nickname={user.nickname}
                avatar={user.avatar}
                apiBase={apiBase}
                showFollow={true}
                showBackArrow={false} // NO back arrow
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
