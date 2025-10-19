import { useState, useEffect } from "react";
import axios from "axios";

interface CreatePostModalProps {
  onClose: () => void;
  onPostCreated: () => void;
}

interface Group {
  group_id: number;
  title: string;
}

export default function CreatePostModal({
  onClose,
  onPostCreated,
}: CreatePostModalProps) {
  const [content, setContent] = useState("");
  const [privacy, setPrivacy] = useState("public");
  const [groupId, setGroupId] = useState<number | null>(null);
  const [groups, setGroups] = useState<Group[]>([]);
  const [media, setMedia] = useState<File | null>(null);
  const [loading, setLoading] = useState(false);

  const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL;

  // Fetch groups
  useEffect(() => {
    const fetchGroups = async () => {
      try {
        const token = localStorage.getItem("token");
        if (!token) return;

        const res = await axios.get(`${API_BASE}/groups/my`, {
          headers: { Authorization: `Bearer ${token}` },
        });

        setGroups(res.data);

        if (res.data.length === 1) {
          setGroupId(res.data[0].group_id);
        }
      } catch (err) {
        console.error("Error fetching groups:", err);
      }
    };
    fetchGroups();
  }, [API_BASE]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!content.trim()) return;

    if (privacy === "private" && !groupId) {
      alert("Please select a group to post to.");
      return;
    }

    const token = localStorage.getItem("token");
    const formData = new FormData();
    formData.append("content", content);
    formData.append("privacy", privacy);
    if (privacy === "private" && groupId) {
      formData.append("group_id", String(groupId));
    }
    if (media) formData.append("media", media);

    try {
      setLoading(true);
      await axios.post(`${API_BASE}/posts`, formData, {
        headers: { Authorization: `Bearer ${token}` },
      });
      setContent("");
      setPrivacy("public");
      setGroupId(null);
      setMedia(null);
      onClose();
      onPostCreated();
    } catch (err) {
      console.error("Error creating post:", err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-40 backdrop-blur-sm flex justify-center items-center z-50">
      <div className="bg-gradient-to-br from-white to-gray-50 rounded-2xl shadow-2xl w-full max-w-md p-6 relative">
        {/* Title */}
        <h2 className="text-2xl font-extrabold text-gray-900 mb-5 border-b pb-3">
          Create Post
        </h2>

        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Content */}
          <textarea
            value={content}
            onChange={(e) => setContent(e.target.value)}
            placeholder="What's on your mind?"
            className="w-full border border-gray-300 bg-white text-gray-900 rounded-lg p-3 resize-none focus:ring-2 focus:ring-indigo-500 focus:outline-none"
            rows={3}
            required
          />

          {/* Privacy */}
          <select
            value={privacy}
            onChange={(e) => setPrivacy(e.target.value)}
            className="w-full border border-gray-300 bg-white text-gray-900 rounded-lg p-3 focus:ring-2 focus:ring-indigo-500 focus:outline-none"
          >
            <option value="public">🌍 Public</option>
            <option value="almost_private">👥 Followers Only</option>
            <option value="private">🔒 Private</option>
          </select>

          {/* Group selection */}
          {privacy === "private" && (
            <select
              value={groupId ?? ""}
              onChange={(e) => setGroupId(Number(e.target.value))}
              className="w-full border border-gray-300 bg-white text-gray-900 rounded-lg p-3 focus:ring-2 focus:ring-indigo-500 focus:outline-none"
              required
            >
              <option value="">Select a group</option>
              {groups.map((group) => (
                <option key={group.group_id} value={group.group_id}>
                  {group.title}
                </option>
              ))}
            </select>
          )}

          {/* Media */}
          <input
            type="file"
            accept="image/*,video/*"
            onChange={(e) =>
              setMedia(e.target.files ? e.target.files[0] : null)
            }
            className="w-full text-gray-700 file:mr-4 file:py-2 file:px-4 file:rounded-lg file:border-0 file:text-sm file:font-semibold file:bg-indigo-600 file:text-white hover:file:bg-indigo-700"
          />

          {/* Buttons */}
          <div className="flex justify-end space-x-2 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 rounded-lg bg-gray-200 text-gray-800 hover:bg-gray-300 transition"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading}
              className="px-4 py-2 rounded-lg bg-gradient-to-r from-indigo-600 to-blue-600 text-white font-semibold shadow hover:from-indigo-700 hover:to-blue-700 transition disabled:opacity-50"
            >
              {loading ? "Posting..." : "Post"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
