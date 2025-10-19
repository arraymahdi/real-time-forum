"use client";

import { useEffect, useState, useRef } from "react";
import axios from "axios";

interface GroupRequest {
  group_id: number;
  target_user_id: number;
  requester_name: string;
  group_name: string;
  request_type: "invitations" | "requests";
}

interface FollowRequest {
  request_id: number;
  follower_name: string;
}

export default function PendingRequestsPage() {
  const [groupRequests, setGroupRequests] = useState<GroupRequest[]>([]);
  const [followRequests, setFollowRequests] = useState<FollowRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [processing, setProcessing] = useState<Set<string>>(new Set());

  const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL || "";

  const audioRef = useRef<HTMLAudioElement | null>(null);
  const previousCountsRef = useRef({ group: 0, follow: 0 });
  const isFirstFetchRef = useRef(true);

  useEffect(() => {
    audioRef.current = new Audio("/notification.mp3");
    audioRef.current.volume = 0.5;
  }, []);

  const playSound = () => {
    try {
      audioRef.current?.play().catch(err => {
        console.debug("Sound play prevented:", err);
      });
    } catch (err) {
      console.error("Failed to play sound", err);
    }
  };

  const startProcessing = (key: string) =>
    setProcessing((prev) => new Set(prev).add(key));
  
  const stopProcessing = (key: string) =>
    setProcessing((prev) => {
      const copy = new Set(prev);
      copy.delete(key);
      return copy;
    });

  const getToken = () => {
    const token = localStorage.getItem("token");
    if (!token) throw new Error("No token found in localStorage");
    return token.startsWith("Bearer ") ? token : `Bearer ${token}`;
  };

  const fetchRequests = async () => {
    setError(null);

    try {
      const token = getToken();
      const headers = { Authorization: token };

      const [invitationRes, requestRes, followRes] = await Promise.all([
        axios.get(`${API_BASE}/groups/pending?type=invitations`, { headers }),
        axios.get(`${API_BASE}/groups/pending?type=requests`, { headers }),
        axios.get(`${API_BASE}/api/follow-requests`, { headers }),
      ]);

      const invitations: GroupRequest[] = Array.isArray(invitationRes.data)
        ? invitationRes.data.map((r: any) => ({
            group_id: r.group_id,
            target_user_id: r.user_id,
            requester_name: r.user_name,
            group_name: r.group_title,
            request_type: "invitations",
          }))
        : [];

      const requests: GroupRequest[] = Array.isArray(requestRes.data)
        ? requestRes.data.map((r: any) => ({
            group_id: r.group_id,
            target_user_id: r.user_id,
            requester_name: r.user_name,
            group_name: r.group_title,
            request_type: "requests",
          }))
        : [];

      const follows: FollowRequest[] = Array.isArray(followRes.data)
        ? followRes.data.map((r: any) => ({
            request_id: r.request_id,
            follower_name: r.follower_name,
          }))
        : [];

      const newGroupCount = invitations.length + requests.length;
      const newFollowCount = follows.length;

      if (!isFirstFetchRef.current) {
        if (
          newGroupCount > previousCountsRef.current.group ||
          newFollowCount > previousCountsRef.current.follow
        ) {
          playSound();
        }
      }

      previousCountsRef.current = {
        group: newGroupCount,
        follow: newFollowCount,
      };
      isFirstFetchRef.current = false;

      setGroupRequests([...invitations, ...requests]);
      setFollowRequests(follows);
      setLoading(false);
    } catch (err: any) {
      console.error("Error fetching requests:", err);
      
      if (err.response?.status === 401) {
        const token = localStorage.getItem("token");
        if (!token) {
          setLoading(false);
          return;
        }
        setError("Session expired. Please log in again.");
      } else {
        setError(err.message || "Failed to fetch requests");
      }
      
      setGroupRequests([]);
      setFollowRequests([]);
      setLoading(false);
    }
  };

  const handleGroupRequest = async (
    groupId: number,
    targetUserId: number,
    action: "accept" | "decline",
    requestType: "invitations" | "requests"
  ) => {
    const key = `group-${groupId}-${targetUserId}-${requestType}`;
    startProcessing(key);

    try {
      const token = getToken();
      const headers = { Authorization: token };

      const backendType = requestType === "invitations" ? "invitation" : "join_request";
      const backendAction = action === "accept" ? "accept" : "reject";

      const body =
        backendType === "invitation"
          ? { group_id: groupId, action: backendAction, request_type: backendType }
          : { group_id: groupId, target_user_id: targetUserId, action: backendAction, request_type: backendType };

      await axios.post(`${API_BASE}/groups/respond`, body, { headers });

      setGroupRequests((prev) =>
        prev.filter(
          (r) =>
            !(
              r.group_id === groupId &&
              r.target_user_id === targetUserId &&
              r.request_type === requestType
            )
        )
      );

      previousCountsRef.current.group--;
      playSound();

      window.dispatchEvent(
        new CustomEvent("request-updated", { 
          detail: { type: "group", groupId, action } 
        })
      );
    } catch (err: any) {
      console.error("Error handling group request:", err);
      setError(`Failed to ${action} request: ${err.message}`);
    } finally {
      stopProcessing(key);
    }
  };

  const handleFollowRequest = async (requestId: number, action: "accept" | "decline") => {
    const key = `follow-${requestId}`;
    startProcessing(key);

    try {
      const token = getToken();
      const headers = { Authorization: token };

      await axios.post(
        `${API_BASE}/api/follow-requests/handle`,
        { request_id: requestId, action },
        { headers }
      );

      setFollowRequests((prev) => prev.filter((r) => r.request_id !== requestId));
      previousCountsRef.current.follow--;
      playSound();

      window.dispatchEvent(
        new CustomEvent("request-updated", { 
          detail: { type: "follow", requestId, action } 
        })
      );
    } catch (err: any) {
      console.error("Error handling follow request:", err);
      setError(`Failed to ${action} request: ${err.message}`);
    } finally {
      stopProcessing(key);
    }
  };

  useEffect(() => {
    fetchRequests();
    
    const interval = setInterval(() => {
      if (localStorage.getItem("token")) {
        fetchRequests();
      }
    }, 15000);
    
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    const handleStorageChange = (e: StorageEvent) => {
      if (e.key === "token" && !e.newValue) {
        setGroupRequests([]);
        setFollowRequests([]);
      }
    };

    window.addEventListener("storage", handleStorageChange);
    return () => window.removeEventListener("storage", handleStorageChange);
  }, []);

  return (
    <div className="max-w-2xl mx-auto p-6">
      <h1 className="text-2xl font-bold text-gray-800 mb-6">Pending Requests</h1>

      {loading && <p className="text-gray-500">Loading requests...</p>}
      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded mb-4">
          {error}
        </div>
      )}

      <section className="mb-8">
        <h2 className="text-lg font-semibold text-gray-700 mb-3">
          Group Requests ({groupRequests.length})
        </h2>
        {groupRequests.length === 0 ? (
          <p className="text-gray-500 text-sm">No group requests</p>
        ) : (
          <div className="space-y-3">
            {groupRequests.map((req) => {
              const key = `group-${req.group_id}-${req.target_user_id}-${req.request_type}`;
              return (
                <div key={key} className="p-4 bg-white shadow rounded-md border flex justify-between items-center">
                  <p className="text-gray-800 text-sm font-medium">
                    <span className="font-semibold">{req.requester_name}</span>{" "}
                    {req.request_type === "invitations" ? "invited you to" : "requested to join"}{" "}
                    <span className="font-semibold">{req.group_name}</span>
                  </p>
                  <div className="flex space-x-2">
                    <button
                      disabled={processing.has(key)}
                      onClick={() => handleGroupRequest(req.group_id, req.target_user_id, "accept", req.request_type)}
                      className={`px-3 py-1 text-white text-sm rounded ${
                        processing.has(key) ? "bg-green-300 cursor-not-allowed" : "bg-green-500 hover:bg-green-600"
                      }`}
                    >
                      {processing.has(key) ? "Processing..." : "Accept"}
                    </button>
                    <button
                      disabled={processing.has(key)}
                      onClick={() => handleGroupRequest(req.group_id, req.target_user_id, "decline", req.request_type)}
                      className={`px-3 py-1 text-white text-sm rounded ${
                        processing.has(key) ? "bg-red-300 cursor-not-allowed" : "bg-red-500 hover:bg-red-600"
                      }`}
                    >
                      {processing.has(key) ? "Processing..." : "Decline"}
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </section>

      <section>
        <h2 className="text-lg font-semibold text-gray-700 mb-3">
          Follow Requests ({followRequests.length})
        </h2>
        {followRequests.length === 0 ? (
          <p className="text-gray-500 text-sm">No follow requests</p>
        ) : (
          <div className="space-y-3">
            {followRequests.map((req) => {
              const key = `follow-${req.request_id}`;
              return (
                <div key={key} className="p-4 bg-white shadow rounded-md border flex justify-between items-center">
                  <p className="text-gray-800 text-sm font-medium">
                    <span className="font-semibold">{req.follower_name}</span> wants to follow you
                  </p>
                  <div className="flex space-x-2">
                    <button
                      disabled={processing.has(key)}
                      onClick={() => handleFollowRequest(req.request_id, "accept")}
                      className={`px-3 py-1 text-white text-sm rounded ${
                        processing.has(key) ? "bg-green-300 cursor-not-allowed" : "bg-green-500 hover:bg-green-600"
                      }`}
                    >
                      {processing.has(key) ? "Processing..." : "Accept"}
                    </button>
                    <button
                      disabled={processing.has(key)}
                      onClick={() => handleFollowRequest(req.request_id, "decline")}
                      className={`px-3 py-1 text-white text-sm rounded ${
                        processing.has(key) ? "bg-red-300 cursor-not-allowed" : "bg-red-500 hover:bg-red-600"
                      }`}
                    >
                      {processing.has(key) ? "Processing..." : "Decline"}
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </section>
    </div>
  );
}