import { useEffect, useState } from "react";
import { useRouter } from "next/router";
import UserHeader from "@/components/UserHeader";

interface Post {
  post_id: number;
  user_id: number;
  nickname?: string;
  content: string;
  media?: string;
  privacy?: string;
  created_at: string;
  avatar?: string;
}

interface Comment {
  comment_id: number;
  post_id: number;
  user_id: number;
  nickname?: string;
  content: string;
  created_at: string;
  avatar?: string;
}

const apiBase =
  typeof window !== "undefined"
    ? process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8088"
    : "";

export default function PostDetailsPage() {
  const router = useRouter();
  const { id } = router.query;

  const [post, setPost] = useState<Post | null>(null);
  const [comments, setComments] = useState<Comment[]>([]);
  const [newComment, setNewComment] = useState("");
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);

  const buildMediaUrl = (path?: string) => {
    if (!path) return "";
    const cleanPath = path.replace(/\\/g, "/");
    if (cleanPath.startsWith("http")) return cleanPath;
    const base = apiBase.endsWith("/") ? apiBase.slice(0, -1) : apiBase;
    const p = cleanPath.startsWith("/") ? cleanPath : `/${cleanPath}`;
    return `${base}${p}`;
  };

  const fetchPost = async () => {
    if (!id) return;
    const token = localStorage.getItem("token") || "";
    try {
      const res = await fetch(`${apiBase}/post/${id}`, {
        headers: { Authorization: token },
      });
      if (res.status === 401) {
        router.push("/signin");
        return;
      }
      if (!res.ok) {
        console.error("Failed to fetch post:", res.status, await res.text());
        setPost(null);
        return;
      }
      const data = await res.json();
      setPost(data);
    } catch (err) {
      console.error("Error fetching post:", err);
      setPost(null);
    }
  };

  const fetchComments = async () => {
    if (!id) return;
    try {
      const res = await fetch(`${apiBase}/comments/all?post_id=${id}`);
      if (!res.ok) {
        console.error("Failed to fetch comments:", res.status, await res.text());
        setComments([]);
        return;
      }
      const data = await res.json();
      setComments(Array.isArray(data) ? data : []);
    } catch (err) {
      console.error("Error fetching comments:", err);
      setComments([]);
    }
  };

  useEffect(() => {
    if (!id) return;
    setLoading(true);
    Promise.all([fetchPost(), fetchComments()]).finally(() =>
      setLoading(false)
    );
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  const handleCommentSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newComment.trim() || !id) return;

    const token = localStorage.getItem("token");
    if (!token) {
      alert("You must be signed in to comment.");
      router.push("/signin");
      return;
    }

    setSubmitting(true);
    try {
      const res = await fetch(`${apiBase}/comments?post_id=${id}`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token,
        },
        body: JSON.stringify({
          post_id: Number(id),
          content: newComment,
        }),
      });

      if (res.status === 401) {
        alert("Unauthorized — please sign in again.");
        router.push("/signin");
        return;
      }

      if (!res.ok) {
        const text = await res.text();
        console.error("Failed to post comment:", res.status, text);
        alert("Failed to post comment. See console for details.");
        return;
      }

      setNewComment("");
      await fetchComments();
    } catch (err) {
      console.error("Error posting comment:", err);
      alert("Network or server error while posting comment.");
    } finally {
      setSubmitting(false);
    }
  };

  if (loading) {
    return (
      <div className="max-w-2xl mx-auto p-6">
        <p className="text-gray-500 text-center">Loading post...</p>
      </div>
    );
  }

  if (!post) {
    return (
      <div className="max-w-2xl mx-auto p-6">
        <p className="text-gray-500 text-center">Post not found.</p>
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto p-6">
      {/* Post card */}
      <div className="bg-white rounded-xl shadow border border-gray-200 p-5 mb-6">
        <UserHeader
          userId={post.user_id}
          nickname={post.nickname}
          avatar={post.avatar}
          showFollow={true}
          apiBase={apiBase}
          showBackArrow={true}
          onBack={() => router.back()}
        />
        <div className="text-xs text-gray-500 mt-1">
          {new Date(post.created_at).toLocaleString()}
        </div>

        <p className="mt-4 text-gray-800 whitespace-pre-line">{post.content}</p>

        {post.media && (
          <div className="mt-4 rounded-lg overflow-hidden">
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

      {/* Comments */}
      <div className="bg-white rounded-xl shadow border border-gray-200 p-5">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-gray-900">Comments</h3>
          <div className="text-sm text-gray-500">
            {comments.length} {comments.length === 1 ? "comment" : "comments"}
          </div>
        </div>

        <div className="space-y-4 mb-4">
          {comments.length === 0 ? (
            <div className="text-gray-500">No comments yet — be the first.</div>
          ) : (
            comments.map((c) => (
              <div key={c.comment_id} className="flex gap-3">
                <UserHeader
                  userId={c.user_id}
                  nickname={c.nickname}
                  avatar={c.avatar}
                  showFollow={false}
                  apiBase={apiBase}
                />
                <div className="flex-1">
                  <div className="text-xs text-gray-400">
                    {new Date(c.created_at).toLocaleString()}
                  </div>
                  <div className="text-gray-700 mt-1">{c.content}</div>
                </div>
              </div>
            ))
          )}
        </div>

        {/* Add comment form */}
        <form onSubmit={handleCommentSubmit} className="mt-2">
          <div className="flex gap-3">
            <input
              value={newComment}
              onChange={(e) => setNewComment(e.target.value)}
              placeholder="Write a comment..."
              className="flex-1 border border-gray-300 rounded-lg px-3 py-2 
                        focus:outline-none focus:ring-2 focus:ring-blue-500 
                        text-gray-800 placeholder-gray-400"
            />
            <button
              type="submit"
              disabled={submitting}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition disabled:opacity-50"
            >
              {submitting ? "Posting..." : "Post"}
            </button>
          </div>
        </form>

      </div>
    </div>
  );
}
