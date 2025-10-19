"use client";

import { useState } from "react";
import api from "../lib/axiosClient";

interface Props {
  onClose: () => void;
  onCreated: (group: any) => void;
}

export default function GroupCreateModal({ onClose, onCreated }: Props) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);

    try {
      const res = await api.post("/groups/create", { title, description });
      onCreated(res.data);
    } catch (err) {
      console.error(err);
      alert("Error creating group");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50">
      <div className="bg-white rounded-2xl shadow-2xl w-full max-w-md animate-fadeIn">
        {/* Header */}
        <div className="flex justify-between items-center border-b px-6 py-4">
          <h2 className="text-lg font-semibold text-gray-800">Create New Group</h2>
          <button
            onClick={onClose}
            className="text-gray-500 hover:text-gray-800 transition-colors"
          >
            ✕
          </button>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="px-6 py-5 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-600 mb-1">
              Group Title
            </label>
            <input
              type="text"
              placeholder="Enter group name"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="w-full border rounded-lg px-3 py-2 text-gray-900 placeholder-gray-400 bg-white focus:ring-2 focus:ring-blue-500 focus:outline-none"
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-600 mb-1">
              Description
            </label>
            <textarea
              placeholder="Enter a short description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
              className="w-full border rounded-lg px-3 py-2 text-gray-900 placeholder-gray-400 bg-white focus:ring-2 focus:ring-blue-500 focus:outline-none"
            />
          </div>

          <div className="flex justify-end gap-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 rounded-lg border text-gray-600 hover:bg-gray-100 transition"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading}
              className="px-4 py-2 rounded-lg bg-blue-600 text-white hover:bg-blue-700 transition disabled:opacity-50"
            >
              {loading ? "Creating..." : "Create"}
            </button>
          </div>
        </form>
      </div>

      {/* Animation */}
      <style jsx>{`
        @keyframes fadeIn {
          from {
            opacity: 0;
            transform: translateY(-10px) scale(0.95);
          }
          to {
            opacity: 1;
            transform: translateY(0) scale(1);
          }
        }
        .animate-fadeIn {
          animation: fadeIn 0.25s ease-out;
        }
      `}</style>
    </div>
  );
}
