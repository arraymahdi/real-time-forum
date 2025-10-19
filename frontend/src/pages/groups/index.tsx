"use client";

import { useEffect, useState } from "react";
import { useAuth } from "../../context/AuthContext";
import api from "../../lib/axiosClient";
import GroupCreateModal from "../../components/GroupCreateModal";
import { useRouter } from "next/navigation";
import { Users } from "lucide-react";

type MembershipStatus = "accepted" | "pending" | "invited" | "none";

interface Group {
  group_id: number;
  title: string;
  description?: string;
  creator_id: number;
  creator_name?: string;
  created_at: string;
  member_count: number;
  membership_status?: MembershipStatus;
}

export default function GroupsPage() {
  const { isSignedIn } = useAuth();
  const [groups, setGroups] = useState<Group[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const router = useRouter();

  const mergeStatuses = (groupsData: Group[], statuses: Record<string, string>) =>
    groupsData.map((g) => ({
      ...g,
      membership_status: (statuses[String(g.group_id)] as MembershipStatus) || "none",
    }));

  useEffect(() => {
    if (!isSignedIn) return;

    let mounted = true;
    async function load() {
      setLoading(true);
      try {
        const [groupsRes, statusRes] = await Promise.all([
          api.get("/groups/browse"),
          api.get("/groups/membership-status"),
        ]);
        if (!mounted) return;
        const statuses: Record<string, string> = statusRes.data || {};
        const merged = mergeStatuses(groupsRes.data, statuses);
        setGroups(merged);
      } catch (err) {
        console.error("Error loading groups or statuses", err);
      } finally {
        if (mounted) setLoading(false);
      }
    }
    load();
    return () => {
      mounted = false;
    };
  }, [isSignedIn]);

  const refreshMembershipStatuses = async () => {
    try {
      const statusRes = await api.get("/groups/membership-status");
      const statuses: Record<string, string> = statusRes.data || {};
      setGroups((prev) => mergeStatuses(prev, statuses));
    } catch (err) {
      console.error("Failed to refresh membership statuses", err);
    }
  };

  async function requestJoin(groupId: number) {
    try {
      setGroups((prev) =>
        prev.map((g) =>
          g.group_id === groupId ? { ...g, membership_status: "pending" } : g
        )
      );

      await api.post("/groups/request", { group_id: groupId });
      await refreshMembershipStatuses();

      const statusRes = await api.get("/groups/membership-status");
      const newStatus = (statusRes.data || {})[String(groupId)];
      if (newStatus === "accepted") router.push(`/groups/${groupId}`);
    } catch (err: any) {
      console.error("Error sending join request", err);
      setGroups((prev) =>
        prev.map((g) =>
          g.group_id === groupId ? { ...g, membership_status: "none" } : g
        )
      );
      const msg =
        err?.response?.data?.message ||
        "Error sending join request (server or network)";
      alert(msg);
    }
  }

  const handleCreated = (newGroup: any) => {
    const groupWithStatus: Group = {
      ...newGroup,
      membership_status: "accepted",
    };
    setGroups((prev) => [groupWithStatus, ...prev]);
    setShowCreateModal(false);
    router.push(`/groups/${newGroup.group_id}`);
  };

  const goToGroup = (groupId: number) => router.push(`/groups/${groupId}`);

  return (
    <div className="p-8 max-w-6xl mx-auto">
      {/* Header */}
      <div className="flex justify-between items-center mb-10">
        <h1 className="text-4xl font-extrabold text-gray-900 tracking-tight">
          Community Groups
        </h1>
        {isSignedIn && (
          <button
            className="px-5 py-2.5 bg-gradient-to-r from-indigo-600 to-blue-600 text-white font-semibold rounded-xl shadow-md hover:from-indigo-700 hover:to-blue-700 transition"
            onClick={() => setShowCreateModal(true)}
          >
            + Create Group
          </button>
        )}
      </div>

      {/* Groups List */}
      {loading ? (
        <p className="text-gray-600">Loading groups...</p>
      ) : groups.length === 0 ? (
        <div className="text-center py-16 bg-gray-50 rounded-xl shadow-inner">
          <p className="text-lg text-gray-600">No groups found.</p>
          <p className="text-sm text-gray-500 mt-2">
            Be the first to create a group and start building a community!
          </p>
        </div>
      ) : (
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-6">
          {groups.map((group) => (
            <div
              key={group.group_id}
              className="p-6 rounded-xl bg-white border shadow-sm hover:shadow-lg transition group"
            >
              {/* Group Icon + Info */}
              <div className="flex items-start space-x-4 min-w-0">
                <div className="flex-shrink-0 p-3 bg-gradient-to-br from-blue-500 to-indigo-600 rounded-full shadow text-white">
                  <Users className="w-6 h-6" />
                </div>
                <div className="min-w-0 flex-1">
                  <h2 className="text-lg font-semibold text-gray-900 group-hover:text-indigo-600 transition break-words whitespace-normal leading-snug">
                    {group.title}
                  </h2>
                  {group.description && (
                    <p className="text-sm text-gray-600 mt-1 break-words whitespace-normal">
                      {group.description}
                    </p>
                  )}
                </div>
              </div>

              {/* Stats */}
              <div className="mt-4 flex items-center justify-between">
                <span className="text-xs font-medium bg-gray-100 text-gray-700 px-2 py-1 rounded-full">
                  {group.member_count} members
                </span>
                <span className="text-xs text-gray-400">
                  {new Date(group.created_at).toLocaleDateString()}
                </span>
              </div>

              {/* Action */}
              <div className="mt-5">
                {group.membership_status === "accepted" ? (
                  <button
                    className="w-full px-4 py-2 bg-green-600 text-white rounded-lg shadow hover:bg-green-700 transition"
                    onClick={() => goToGroup(group.group_id)}
                  >
                    Joined
                  </button>
                ) : group.membership_status === "pending" ? (
                  <button
                    className="w-full px-4 py-2 bg-gray-400 text-white rounded-lg shadow cursor-not-allowed"
                    disabled
                  >
                    Requested
                  </button>
                ) : group.membership_status === "invited" ? (
                  <button
                    className="w-full px-4 py-2 bg-purple-500 text-white rounded-lg shadow hover:bg-purple-600 transition"
                    onClick={() => router.push("/requests")}
                  >
                    Invited – View
                  </button>
                ) : (
                  <button
                    className="w-full px-4 py-2 bg-yellow-500 text-white rounded-lg shadow hover:bg-yellow-600 transition"
                    onClick={() => requestJoin(group.group_id)}
                  >
                    Request to Join
                  </button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Modal */}
      {showCreateModal && (
        <GroupCreateModal
          onClose={() => setShowCreateModal(false)}
          onCreated={handleCreated}
        />
      )}
    </div>
  );
}
