"use client";

import { useEffect, useState } from "react";
import { useAuth } from "../context/AuthContext";
import { useRouter } from "next/router";
import UserPostsModal from "@/components/UserPostsModal";
import FollowersModal from "@/components/FollowersModal";
import FollowingModal from "@/components/FollowingModal";

interface UserProfile {
  id: number;
  email: string;
  first_name: string;
  last_name: string;
  date_of_birth: string;
  avatar: string;
  nickname: string;
  about_me: string;
  profile_type: string;
}

const apiBase =
  typeof window !== "undefined"
    ? process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8088"
    : "";

export default function ProfilePage() {
  const { isSignedIn, loading: authLoading } = useAuth();
  const router = useRouter();
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [showPostsModal, setShowPostsModal] = useState(false);
  const [followersCount, setFollowersCount] = useState(0);
  const [followingCount, setFollowingCount] = useState(0);
  const [showFollowersModal, setShowFollowersModal] = useState(false);
  const [showFollowingModal, setShowFollowingModal] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const buildAvatarUrl = (avatar: string | null | undefined) => {
    if (!avatar) return "";
    const cleanPath = avatar.replace(/\\/g, "/");
    if (cleanPath.startsWith("http")) return cleanPath;
    const base = apiBase.endsWith("/") ? apiBase.slice(0, -1) : apiBase;
    const path = cleanPath.startsWith("/") ? cleanPath : `/${cleanPath}`;
    return `${base}${path}`;
  };

  const fetchProfile = async () => {
    const token = localStorage.getItem("token");
    if (!token) {
      router.push("/signin");
      return;
    }
    try {
      const res = await fetch(`${apiBase}/user/profile/details`, {
        headers: { Authorization: token },
      });
      if (res.ok) {
        const data = await res.json();
        setProfile(data);
      } else {
        setError("Failed to load profile.");
      }
    } catch (err) {
      console.error("Error fetching profile:", err);
      setError("Failed to load profile.");
    } finally {
      setLoading(false);
    }
  };

  const fetchFollowStats = async () => {
    const token = localStorage.getItem("token");
    if (!token) return;
    try {
      const [followersRes, followingRes] = await Promise.all([
        fetch(`${apiBase}/followers`, { headers: { Authorization: token } }),
        fetch(`${apiBase}/following`, { headers: { Authorization: token } }),
      ]);

      if (followersRes.ok) {
        const followersData = await followersRes.json();
        setFollowersCount(Array.isArray(followersData) ? followersData.length : 0);
      }

      if (followingRes.ok) {
        const followingData = await followingRes.json();
        setFollowingCount(Array.isArray(followingData) ? followingData.length : 0);
      }
    } catch (err) {
      console.error("Error fetching follow stats:", err);
      setError("Failed to load followers/following data.");
    }
  };

  useEffect(() => {
    if (authLoading) return; // wait until auth state is loaded
    if (!isSignedIn) {
      router.push("/signin");
    } else {
      fetchProfile();
      fetchFollowStats();
    }
  }, [isSignedIn, authLoading]);

  const handleSave = async () => {
    if (!profile) return;
    setSaving(true);
    setMessage(null);
    setError(null);
    const token = localStorage.getItem("token");
    try {
      const res = await fetch(`${apiBase}/user/profile/update`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({
          first_name: profile.first_name,
          last_name: profile.last_name,
          date_of_birth: profile.date_of_birth,
          nickname: profile.nickname,
          about_me: profile.about_me,
          profile_type: profile.profile_type,
        }),
      });
      if (res.ok) {
        fetchProfile();
        setMessage("Profile updated successfully!");
      } else {
        setError("Update failed.");
      }
    } catch (err) {
      console.error("Update error:", err);
      setError("Update failed.");
    } finally {
      setSaving(false);
    }
  };

  const handleAvatarUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    if (!e.target.files || e.target.files.length === 0) return;
    const file = e.target.files[0];
    const token = localStorage.getItem("token");
    const formData = new FormData();
    formData.append("avatar", file);
    setMessage(null);
    setError(null);
    try {
      const res = await fetch(`${apiBase}/upload-avatar`, {
        method: "POST",
        headers: { Authorization: token || "" },
        body: formData,
      });
      if (res.ok) {
        fetchProfile();
        setMessage("Avatar uploaded successfully!");
      } else {
        setError("Avatar upload failed.");
      }
    } catch (err) {
      console.error("Upload error:", err);
      setError("Avatar upload failed.");
    }
  };

  if (loading) return <p className="p-6">Loading profile...</p>;
  if (!profile) return <p className="p-6">Profile not found.</p>;

  const token = localStorage.getItem("token");

  return (
    <div className="min-h-screen bg-gray-50 py-10">
      <div className="max-w-2xl mx-auto bg-white rounded-xl shadow-lg p-8">
        <h1 className="text-3xl font-extrabold mb-8 text-center text-gray-900">
          My Profile
        </h1>

        {/* Avatar */}
        <div className="flex flex-col items-center mb-6">
          <div className="relative w-32 h-32 rounded-full overflow-hidden ring-4 ring-blue-100 shadow-md">
            {profile.avatar ? (
              <img
                src={buildAvatarUrl(profile.avatar)}
                alt="Avatar"
                className="w-full h-full object-cover"
              />
            ) : (
              <div className="w-full h-full flex items-center justify-center bg-gradient-to-r from-blue-500 to-indigo-500 text-white text-3xl font-bold">
                {profile.nickname ? profile.nickname[0].toUpperCase() : "U"}
              </div>
            )}
          </div>
          <p className="mt-4 text-lg font-semibold text-gray-800">
            {profile.nickname}
          </p>
          <p className="text-sm text-gray-500">{profile.email}</p>
          <input
            type="file"
            accept="image/*"
            onChange={handleAvatarUpload}
            className="mt-4 text-sm"
          />
        </div>

        {/* Followers / Following */}
        <div className="flex justify-center space-x-12 mb-8">
          <div
            className="text-center cursor-pointer"
            onClick={() => setShowFollowersModal(true)}
          >
            <p className="text-2xl font-bold text-blue-600">{followersCount}</p>
            <p className="text-sm text-gray-600">Followers</p>
          </div>
          <div
            className="text-center cursor-pointer"
            onClick={() => setShowFollowingModal(true)}
          >
            <p className="text-2xl font-bold text-blue-600">{followingCount}</p>
            <p className="text-sm text-gray-600">Following</p>
          </div>
        </div>

        {/* Profile Form */}
        <div className="space-y-4">
          <input
            type="text"
            value={profile.first_name}
            onChange={(e) =>
              setProfile({ ...profile, first_name: e.target.value })
            }
            placeholder="First Name"
            className="w-full p-3 border rounded-lg focus:ring-2 focus:ring-blue-300 focus:outline-none text-gray-800"
          />
          <input
            type="text"
            value={profile.last_name}
            onChange={(e) =>
              setProfile({ ...profile, last_name: e.target.value })
            }
            placeholder="Last Name"
            className="w-full p-3 border rounded-lg focus:ring-2 focus:ring-blue-300 focus:outline-none text-gray-800"
          />
          <input
            type="date"
            value={profile.date_of_birth || ""}
            onChange={(e) =>
              setProfile({ ...profile, date_of_birth: e.target.value })
            }
            className="w-full p-3 border rounded-lg focus:ring-2 focus:ring-blue-300 focus:outline-none text-gray-800"
          />
          <input
            type="text"
            value={profile.nickname}
            onChange={(e) =>
              setProfile({ ...profile, nickname: e.target.value })
            }
            placeholder="Nickname"
            className="w-full p-3 border rounded-lg focus:ring-2 focus:ring-blue-300 focus:outline-none text-gray-800"
          />
          <textarea
            value={profile.about_me}
            onChange={(e) =>
              setProfile({ ...profile, about_me: e.target.value })
            }
            placeholder="About Me"
            className="w-full p-3 border rounded-lg focus:ring-2 focus:ring-blue-300 focus:outline-none min-h-[100px] text-gray-800"
          />
          <select
            value={profile.profile_type || ""}
            onChange={(e) =>
              setProfile({ ...profile, profile_type: e.target.value })
            }
            className="w-full p-3 border rounded-lg focus:ring-2 focus:ring-blue-300 focus:outline-none text-gray-800"
          >
            <option value="">Select Profile Type</option>
            <option value="public">Public</option>
            <option value="private">Private</option>
          </select>
        </div>

        {/* Buttons */}
        <div className="flex flex-col sm:flex-row sm:justify-center gap-4 mt-8">
          <button
            className="bg-gradient-to-r from-blue-600 to-indigo-600 text-white px-6 py-2 rounded-lg hover:opacity-90 transition font-medium"
            onClick={handleSave}
            disabled={saving}
          >
            {saving ? "Saving..." : "Save Changes"}
          </button>
          {token && (
            <button
              className="bg-gray-700 text-white px-6 py-2 rounded-lg hover:bg-gray-800 transition font-medium"
              onClick={() => setShowPostsModal(true)}
            >
              View My Posts
            </button>
          )}
        </div>

        {/* In-page messages */}
        <div className="mt-4 text-center">
          {message && <p className="text-green-600 font-medium">{message}</p>}
          {error && <p className="text-red-600 font-medium">{error}</p>}
        </div>
      </div>

      {/* Modals */}
      {showPostsModal && token && (
        <UserPostsModal
          apiBase={apiBase}
          token={token}
          onClose={() => setShowPostsModal(false)}
        />
      )}
      {showFollowersModal && token && (
        <FollowersModal
          apiBase={apiBase}
          token={token}
          onClose={() => setShowFollowersModal(false)}
        />
      )}
      {showFollowingModal && token && (
        <FollowingModal
          apiBase={apiBase}
          token={token}
          onClose={() => setShowFollowingModal(false)}
        />
      )}
    </div>
  );
}
