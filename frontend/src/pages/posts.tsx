"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/router";
import axios from "axios";
import { Plus } from "lucide-react";
import CreatePostModal from "@/components/CreatePostModal";
import UserHeader from "@/components/UserHeader";

interface Post {
  post_id: number;
  user_id: number;
  nickname?: string;
  avatar?: string;
  content: string;
  media?: string;
  privacy: string;
  created_at: string;
}

export default function PostsPage() {
  const [posts, setPosts] = useState<Post[]>([]);
  const [showForm, setShowForm] = useState(false);
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL;
  const router = useRouter();

  const fetchPosts = async () => {
    try {
      const token = localStorage.getItem("token");
      const res = await axios.get(`${API_BASE}/posts/all`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      setPosts(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error("Error fetching posts:", err);
      setPosts([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchPosts();
  }, []);

  const buildMediaUrl = (path?: string) => {
    if (!path) return "";
    if (path.startsWith("http")) return path;
    const base = API_BASE?.endsWith("/") ? API_BASE.slice(0, -1) : API_BASE;
    const p = path.startsWith("/") ? path : `/${path}`;
    return `${base}${p}`;
  };

  return (
    <div className="max-w-2xl mx-auto p-4 pb-24">
      {/* Posts Feed */}
      <div className="space-y-6">
        {loading ? (
          <p className="text-center text-gray-400">Loading posts...</p>
        ) : posts.length > 0 ? (
          posts.map((post) => (
            <div
              key={post.post_id}
              onClick={() => router.push(`/post/${post.post_id}`)}
              className="bg-white rounded-2xl shadow-sm border border-gray-100 p-5 cursor-pointer hover:shadow-md hover:border-gray-200 transition-all duration-200"
            >
              {/* User header */}
              <UserHeader
                userId={post.user_id}
                nickname={post.nickname}
                avatar={post.avatar}
                showFollow={false}
                apiBase={API_BASE || ""}
              />

              {/* Timestamp */}
              <div className="text-xs text-gray-400 mt-1 mb-3">
                {new Date(post.created_at).toLocaleString()}
              </div>

              {/* Content */}
              <p className="text-gray-800 mb-3 leading-relaxed whitespace-pre-wrap">
                {post.content}
              </p>

              {/* Media */}
              {post.media && (
                <div className="rounded-xl overflow-hidden border border-gray-100">
                  {post.media.match(/\.(mp4|webm|ogg)$/i) ? (
                    <video
                      controls
                      className="w-full rounded-lg max-h-[400px] object-cover"
                    >
                      <source src={buildMediaUrl(post.media)} />
                    </video>
                  ) : (
                    <img
                      src={buildMediaUrl(post.media)}
                      alt="Post media"
                      className="w-full max-h-[400px] object-cover"
                    />
                  )}
                </div>
              )}
            </div>
          ))
        ) : (
          <p className="text-center text-gray-500">No posts available</p>
        )}
      </div>

      {/* Floating Create Button */}
      <button
        onClick={() => setShowForm(true)}
        className="fixed bottom-20 right-6 bg-gradient-to-r from-blue-500 to-indigo-600 text-white rounded-full p-4 shadow-lg hover:scale-110 transform transition duration-200"
      >
        <Plus className="w-6 h-6" />
      </button>

      {/* Create Post Modal */}
      {showForm && (
        <CreatePostModal
          onClose={() => setShowForm(false)}
          onPostCreated={fetchPosts}
        />
      )}
    </div>
  );
}
