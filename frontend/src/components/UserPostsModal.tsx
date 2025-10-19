import { useEffect, useState } from "react";
import { useRouter } from "next/router";
import axios from "axios";
import { X } from "lucide-react";
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

interface UserPostsModalProps {
  apiBase: string;
  token: string;
  onClose: () => void;
}

export default function UserPostsModal({ apiBase, token, onClose }: UserPostsModalProps) {
  const [posts, setPosts] = useState<Post[]>([]);
  const [loading, setLoading] = useState(true);
  const router = useRouter();

  useEffect(() => {
    const fetchUserPosts = async () => {
      setLoading(true);
      try {
        const res = await axios.get(`${apiBase}/posts/mine`, {
          headers: { Authorization: `Bearer ${token}` },
        });
        setPosts(Array.isArray(res.data) ? res.data : []);
      } catch (err) {
        console.error("Error fetching user posts:", err);
        setPosts([]);
      } finally {
        setLoading(false);
      }
    };
    fetchUserPosts();
  }, [apiBase, token]);

  const buildMediaUrl = (path?: string) => {
    if (!path) return "";
    if (path.startsWith("http")) return path;
    const base = apiBase.endsWith("/") ? apiBase.slice(0, -1) : apiBase;
    const p = path.startsWith("/") ? path : `/${path}`;
    return `${base}${p}`;
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex justify-center items-start overflow-auto z-50 p-4">
      <div className="bg-white rounded-2xl w-full max-w-3xl mt-16 p-6 relative shadow-xl">
        {/* Close Button */}
        <button
          className="absolute top-4 right-4 text-gray-500 hover:text-gray-800"
          onClick={onClose}
        >
          <X size={24} />
        </button>

        <h2 className="text-2xl font-bold mb-6 text-center">My Posts</h2>

        {loading ? (
          <p className="text-center text-gray-500">Loading posts...</p>
        ) : posts.length === 0 ? (
          <p className="text-center text-gray-500">No posts yet.</p>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {posts.map((post) => (
              <div
                key={post.post_id}
                onClick={() => router.push(`/post/${post.post_id}`)}
                className="bg-gray-50 rounded-xl shadow-sm border border-gray-200 p-4 cursor-pointer hover:shadow-lg transition duration-200"
              >
                {/* User Header */}
                <UserHeader
                  userId={post.user_id}
                  nickname={post.nickname}
                  avatar={post.avatar}
                  showFollow={false}
                  apiBase={apiBase}
                />

                <div className="text-xs text-gray-500 mt-1 mb-2">
                  {new Date(post.created_at).toLocaleString()}
                </div>

                <p className="text-gray-700 mb-3 line-clamp-3">{post.content}</p>

                {post.media && (
                  <div className="rounded-lg overflow-hidden">
                    {post.media.match(/\.(mp4|webm|ogg)$/i) ? (
                      <video controls className="w-full rounded-lg">
                        <source src={buildMediaUrl(post.media)} />
                      </video>
                    ) : (
                      <img
                        src={buildMediaUrl(post.media)}
                        alt="Post media"
                        className="w-full rounded-lg"
                      />
                    )}
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
