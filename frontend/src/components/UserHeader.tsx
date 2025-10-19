import { useState, useEffect } from "react";
import { useRouter } from "next/router";

interface UserProfile {
  id: number;
  nickname: string;
  profile_type: string;
  avatar?: string;
  first_name?: string;
  last_name?: string;
}

interface FollowStatus {
  is_following: boolean;
  follows_you: boolean;
  follow_request_pending?: boolean;
}

interface UserHeaderProps {
  userId: number;
  nickname?: string;
  avatar?: string;
  showFollow?: boolean;
  apiBase: string;
  showBackArrow?: boolean;
  onBack?: () => void;
}

export default function UserHeader({
  userId,
  nickname: initialNickname,
  avatar: initialAvatar,
  showFollow = false,
  apiBase,
  showBackArrow = false,
  onBack,
}: UserHeaderProps) {
  const router = useRouter();
  const [user, setUser] = useState<UserProfile | null>(null);
  const [followStatus, setFollowStatus] = useState<FollowStatus>({
    is_following: false,
    follows_you: false,
    follow_request_pending: false
  });
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(false);

  const buildMediaUrl = (path?: string) => {
    if (!path) return "";
    const cleanPath = path.replace(/\\/g, "/");
    if (cleanPath.startsWith("http")) return cleanPath;
    const base = apiBase.endsWith("/") ? apiBase.slice(0, -1) : apiBase;
    const p = cleanPath.startsWith("/") ? cleanPath : `/${cleanPath}`;
    return `${base}${p}`;
  };

  const getToken = () => {
    try {
      return localStorage.getItem("token");
    } catch (error) {
      console.error("Error accessing localStorage:", error);
      return null;
    }
  };

  const getCurrentUserId = () => {
    try {
      const id = localStorage.getItem("user_id");
      return id ? parseInt(id) : null;
    } catch (error) {
      console.error("Error getting current user ID:", error);
      return null;
    }
  };

  // Fetch user profile data
  useEffect(() => {
    const fetchUser = async () => {
      try {
        const token = getToken();
        if (!token) {
          console.warn("No token available for fetching user");
          return;
        }

        console.log(`Fetching user data for ID: ${userId}`);

        const res = await fetch(`${apiBase}/user/${userId}`, {
          headers: { 
            Authorization: token.startsWith("Bearer ") ? token : `Bearer ${token}`
          },
        });

        if (res.ok) {
          const data = await res.json();
          console.log("User data received:", data);
          setUser(data);
        } else {
          console.error("Failed to fetch user", res.status, await res.text());
        }
      } catch (err) {
        console.error("Error fetching user:", err);
      }
    };

    // If we don't have user data passed as props, fetch it
    if (!initialNickname || !initialAvatar) {
      fetchUser();
    } else {
      setUser({ 
        id: userId, 
        nickname: initialNickname, 
        avatar: initialAvatar,
        profile_type: "public" // Default assumption, will be updated by follow status call
      });
    }
  }, [userId, initialNickname, initialAvatar, apiBase]);

  // Fetch follow status and check for pending requests
  useEffect(() => {
    const fetchFollowStatus = async () => {
      if (!showFollow || !user) return;
      
      const token = getToken();
      const currentUserId = getCurrentUserId();
      
      if (!token || !currentUserId || currentUserId === userId) {
        setLoading(false);
        return;
      }

      try {
        console.log(`Fetching follow status for user: ${user.nickname}`);
        
        // Fetch follow status
        const res = await fetch(`${apiBase}/user/follow-status?nickname=${encodeURIComponent(user.nickname)}`, {
          headers: { 
            Authorization: token.startsWith("Bearer ") ? token : `Bearer ${token}`
          },
        });

        if (res.ok) {
          const data = await res.json();
          console.log("Follow status received:", data);
          
          let pendingRequest = false;
          
          // If not following, check for pending request
          if (!data.is_following) {
            try {
              const requestRes = await fetch(`${apiBase}/api/follow-request-status?nickname=${encodeURIComponent(user.nickname)}`, {
                headers: { 
                  Authorization: token.startsWith("Bearer ") ? token : `Bearer ${token}`
                },
              });
              
              if (requestRes.ok) {
                const requestData = await requestRes.json();
                pendingRequest = requestData.has_pending_request || false;
                console.log("Pending request status:", pendingRequest);
              }
            } catch (err) {
              console.error("Error checking pending request status:", err);
            }
          }
          
          setFollowStatus({
            is_following: data.is_following || false,
            follows_you: data.follows_you || false,
            follow_request_pending: pendingRequest
          });
        } else {
          console.error("Failed to fetch follow status:", res.status);
        }
      } catch (err) {
        console.error("Error fetching follow status:", err);
      } finally {
        setLoading(false);
      }
    };

    fetchFollowStatus();
  }, [userId, user, showFollow, apiBase]);

  const handleFollowAction = async () => {
    const token = getToken();
    const currentUserId = getCurrentUserId();
    
    if (!token || !user) {
      alert("You must be signed in to follow users.");
      return;
    }

    if (currentUserId === userId) {
      alert("You cannot follow yourself.");
      return;
    }

    // Don't allow action if request is pending
    if (followStatus.follow_request_pending) {
      return;
    }

    setActionLoading(true);

    try {
      if (followStatus.is_following) {
        // Unfollow user
        console.log(`Unfollowing user: ${user.nickname}`);
        
        const res = await fetch(`${apiBase}/unfollow?nickname=${encodeURIComponent(user.nickname)}`, {
          method: "DELETE",
          headers: {
            "Content-Type": "application/json",
            Authorization: token.startsWith("Bearer ") ? token : `Bearer ${token}`,
          },
        });

        if (res.ok) {
          console.log("Successfully unfollowed user");
          setFollowStatus(prev => ({
            ...prev,
            is_following: false,
            follow_request_pending: false
          }));
        } else {
          const errorText = await res.text();
          console.error("Failed to unfollow:", res.status, errorText);
          alert("Failed to unfollow user. Please try again.");
        }
      } else {
        // Follow user (this will handle both public and private profiles on the backend)
        console.log(`Following user: ${user.nickname} (${user.profile_type})`);
        
        const res = await fetch(`${apiBase}/follow?nickname=${encodeURIComponent(user.nickname)}`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: token.startsWith("Bearer ") ? token : `Bearer ${token}`,
          },
        });

        if (res.ok) {
          const result = await res.json();
          console.log("Follow action result:", result);
          
          if (result.message?.includes("Follow request sent")) {
            // Private profile - request sent
            console.log("Follow request sent for private profile");
            setFollowStatus(prev => ({
              ...prev,
              is_following: false,
              follow_request_pending: true
            }));
          } else {
            // Public profile - followed immediately
            console.log("Successfully followed public profile");
            setFollowStatus(prev => ({
              ...prev,
              is_following: true,
              follow_request_pending: false
            }));
          }
        } else {
          const errorText = await res.text();
          console.error("Failed to follow:", res.status, errorText);
          
          if (res.status === 409) {
            // Handle conflict responses
            try {
              const errorData = JSON.parse(errorText);
              if (errorData.message?.includes("Follow request already sent")) {
                setFollowStatus(prev => ({ ...prev, follow_request_pending: true }));
                return; // Don't show alert for this case
              } else if (errorData.message?.includes("Already following")) {
                setFollowStatus(prev => ({ ...prev, is_following: true }));
                return; // Don't show alert for this case
              }
            } catch (parseErr) {
              // If parsing fails, check the text directly
              if (errorText.includes("Already following")) {
                setFollowStatus(prev => ({ ...prev, is_following: true }));
                return;
              } else if (errorText.includes("already sent") || errorText.includes("pending")) {
                setFollowStatus(prev => ({ ...prev, follow_request_pending: true }));
                return;
              }
            }
          }
          
          // Show alert for other errors
          alert("Failed to follow user. Please try again.");
        }
      }
    } catch (err) {
      console.error("Error toggling follow:", err);
      alert("An error occurred. Please try again.");
    } finally {
      setActionLoading(false);
    }
  };

  const getFollowButtonText = () => {
    if (actionLoading) return "Loading...";
    if (followStatus.is_following) return "Following";
    if (followStatus.follow_request_pending) return "Requested";
    return "Follow";
  };

  const getFollowButtonStyle = () => {
    if (followStatus.is_following) {
      return "bg-gray-200 text-gray-700 hover:bg-gray-300";
    }
    if (followStatus.follow_request_pending) {
      return "bg-yellow-500 text-white hover:bg-yellow-600";
    }
    return "bg-blue-600 text-white hover:bg-blue-700";
  };

  if (!user) {
    return (
      <div className="flex items-center gap-2 text-gray-400 text-sm">
        <div className="animate-pulse flex items-center gap-3">
          <div className="w-10 h-10 bg-gray-300 rounded-full"></div>
          <div className="h-4 bg-gray-300 rounded w-24"></div>
        </div>
      </div>
    );
  }

  const currentUserId = getCurrentUserId();
  const isCurrentUser = currentUserId === userId;

  return (
    <div className="flex items-center justify-between">
      <div className="flex items-center gap-3">
        {/* Back arrow */}
        {showBackArrow && (
          <button 
            onClick={onBack || (() => router.push("/posts"))} 
            className="mr-2 p-1 hover:bg-gray-100 rounded"
            title="Go back"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              className="h-6 w-6 text-gray-700 hover:text-gray-900"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
            </svg>
          </button>
        )}

        {/* Avatar */}
        <div className="w-10 h-10 rounded-full overflow-hidden bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center text-sm font-medium text-white border-2 border-gray-200">
          {user.avatar ? (
            <img
              src={buildMediaUrl(user.avatar)}
              alt="User avatar"
              className="w-full h-full object-cover"
              onError={(e) => {
                const target = e.target as HTMLImageElement;
                target.style.display = "none";
                target.nextElementSibling?.classList.remove("hidden");
              }}
            />
          ) : null}
          <div className={`w-full h-full flex items-center justify-center ${user.avatar ? "hidden" : ""}`}>
            {(user.nickname || user.first_name || "U")[0].toUpperCase()}
          </div>
        </div>

        {/* User info */}
        <div className="flex flex-col">
          <span className="font-medium text-gray-900">{user.nickname || "Unknown"}</span>
          {user.profile_type && (
            <span className="text-xs text-gray-500 capitalize">
              {user.profile_type} profile
            </span>
          )}
        </div>
      </div>

      {/* Follow button */}
      {showFollow && !loading && !isCurrentUser && (
        <button
          onClick={handleFollowAction}
          disabled={actionLoading || followStatus.follow_request_pending}
          className={`px-4 py-2 rounded-lg text-sm font-medium transition duration-200 ${getFollowButtonStyle()} ${
            actionLoading || followStatus.follow_request_pending ? "opacity-50 cursor-not-allowed" : ""
          }`}
          title={
            followStatus.is_following 
              ? "Click to unfollow" 
              : followStatus.follow_request_pending 
              ? "Follow request pending" 
              : user.profile_type === "private" 
              ? "Send follow request" 
              : "Follow user"
          }
        >
          {getFollowButtonText()}
        </button>
      )}

      {/* Show "You" indicator for current user */}
      {isCurrentUser && (
        <span className="px-3 py-1 bg-gray-100 text-gray-600 rounded-lg text-sm font-medium">
          You
        </span>
      )}
    </div>
  );
}