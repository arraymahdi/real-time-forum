"use client";

import { useEffect, useState } from "react";
import { useRouter, useParams } from "next/navigation";
import { 
  Users, 
  MessageCircle, 
  Calendar, 
  Plus, 
  ChevronRight,
  ThumbsUp,
  ThumbsDown,
  Clock,
  ExternalLink,
  UserPlus,
  Search
} from "lucide-react";
import { useAuth } from "../../context/AuthContext";
import { useWebSocket } from "../../context/WebSocketContext";
import api from "../../lib/axiosClient";
import CreatePostModal from "../../components/CreatePostModal";

interface Group {
  group_id: number;
  title: string;
  description?: string;
  creator_id: number;
  creator_name?: string;
  created_at: string;
  member_count: number;
  user_role?: string;
}

interface GroupMember {
  user_id: number;
  user_name: string;
  role: string;
  status: string;
  joined_at?: string;
  avatar?: string;
}

interface GroupPost {
  post_id: number;
  user_id: number;
  group_id: number;
  content: string;
  media?: string;
  created_at: string;
  nickname: string;
}

interface Event {
  event_id: number;
  group_id: number;
  creator_id: number;
  title: string;
  description?: string;
  event_time: string;
  created_at: string;
  creator_name?: string;
  user_response?: string;
}

interface User {
  id: number;
  nickname: string;
  email: string;
  avatar?: string;
}

interface EventWithResponses extends Event {
  response_counts: {
    going: number;
    not_going: number;
    total: number;
  };
  responses: Array<{
    user_id: number;
    response: string;
    user_name: string;
  }>;
}

export default function GroupPage() {
  const { isSignedIn } = useAuth();
  const { onlineUsers } = useWebSocket();
  const router = useRouter();
  const params = useParams();
  const groupId = params?.id as string;

  const [group, setGroup] = useState<Group | null>(null);
  const [members, setMembers] = useState<GroupMember[]>([]);
  const [posts, setPosts] = useState<GroupPost[]>([]);
  const [events, setEvents] = useState<Event[]>([]);
  const [eventDetails, setEventDetails] = useState<{[key: number]: EventWithResponses}>({});
  const [loading, setLoading] = useState(true);
  const [showCreatePost, setShowCreatePost] = useState(false);
  const [showCreateEvent, setShowCreateEvent] = useState(false);
  const [showInviteUsers, setShowInviteUsers] = useState(false);
  const [newEventTitle, setNewEventTitle] = useState("");
  const [newEventDescription, setNewEventDescription] = useState("");
  const [newEventTime, setNewEventTime] = useState("");
  const [allUsers, setAllUsers] = useState<User[]>([]);
  const [filteredUsers, setFilteredUsers] = useState<User[]>([]);
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedUsers, setSelectedUsers] = useState<Set<number>>(new Set());
  const [inviteLoading, setInviteLoading] = useState(false);
  const [showSuccessMessage, setShowSuccessMessage] = useState(false);
  const [successText, setSuccessText] = useState("");

  useEffect(() => {
    if (!isSignedIn || !groupId) return;
    loadGroupData();
    loadAllUsers();
  }, [isSignedIn, groupId]);

  useEffect(() => {
    if (showSuccessMessage) {
      const timer = setTimeout(() => setShowSuccessMessage(false), 3000);
      return () => clearTimeout(timer);
    }
  }, [showSuccessMessage]);

  const loadGroupData = async () => {
    setLoading(true);
    try {
      const [groupRes, membersRes, postsRes, eventsRes] = await Promise.all([
        api.get(`/group/${groupId}`),
        api.get(`/group/${groupId}/members`),
        api.get(`/group/${groupId}/posts`),
        api.get(`/group/${groupId}/events`)
      ]);

      setGroup(groupRes.data);
      setMembers(Array.isArray(membersRes.data) ? membersRes.data : []);
      setPosts(Array.isArray(postsRes.data) ? postsRes.data : []);
      setEvents(Array.isArray(eventsRes.data) ? eventsRes.data : []);

      // Load detailed event information for vote counts
      await loadEventDetails(eventsRes.data || []);
    } catch (err: any) {
      console.error("Error loading group data:", err);
      setMembers([]);
      setPosts([]);
      setEvents([]);
      if (err?.response?.status === 403) router.push("/groups");
    } finally {
      setLoading(false);
    }
  };

  const loadAllUsers = async () => {
    try {
      const response = await api.get("/users");
      const users = Array.isArray(response.data) ? response.data : [];
      
      // Filter out users who are already members
      const memberIds = new Set(members.map(m => m.user_id));
      const nonMembers = users.filter(user => !memberIds.has(user.id));
      
      setAllUsers(nonMembers);
      setFilteredUsers(nonMembers);
    } catch (err) {
      console.error("Error loading users:", err);
      setAllUsers([]);
      setFilteredUsers([]);
    }
  };

  // Update filtered users when members change
  useEffect(() => {
    if (allUsers.length > 0 && members.length > 0) {
      const memberIds = new Set(members.map(m => m.user_id));
      const nonMembers = allUsers.filter(user => !memberIds.has(user.id));
      setFilteredUsers(nonMembers);
    }
  }, [allUsers, members]);

  // Filter users based on search query
  useEffect(() => {
    if (!searchQuery.trim()) {
      const memberIds = new Set(members.map(m => m.user_id));
      setFilteredUsers(allUsers.filter(user => !memberIds.has(user.id)));
    } else {
      const memberIds = new Set(members.map(m => m.user_id));
      const filtered = allUsers.filter(user => 
        !memberIds.has(user.id) &&
        (user.nickname.toLowerCase().includes(searchQuery.toLowerCase()) ||
         user.email.toLowerCase().includes(searchQuery.toLowerCase()))
      );
      setFilteredUsers(filtered);
    }
  }, [searchQuery, allUsers, members]);

  const loadEventDetails = async (eventsList: Event[]) => {
    const details: {[key: number]: EventWithResponses} = {};
    
    for (const event of eventsList) {
      try {
        const eventRes = await api.get(`/event/${event.event_id}`);
        details[event.event_id] = eventRes.data;
      } catch (err) {
        console.error(`Error loading event ${event.event_id} details:`, err);
        // Fallback to basic event data
        details[event.event_id] = {
          ...event,
          response_counts: { going: 0, not_going: 0, total: 0 },
          responses: []
        };
      }
    }
    
    setEventDetails(details);
  };

  const handleInviteUsers = async () => {
    if (selectedUsers.size === 0) {
      alert("Please select at least one user to invite");
      return;
    }

    setInviteLoading(true);
    const invitePromises = Array.from(selectedUsers).map(userId =>
      api.post("/groups/invite", {
        group_id: parseInt(groupId),
        invited_user_id: userId
      })
    );

    try {
      await Promise.all(invitePromises);
      setSuccessText(`Successfully invited ${selectedUsers.size} user${selectedUsers.size > 1 ? 's' : ''}`);
      setShowSuccessMessage(true);
      setShowInviteUsers(false);
      setSelectedUsers(new Set());
      setSearchQuery("");
    } catch (err) {
      console.error("Error inviting users:", err);
      alert("Failed to invite some users");
    } finally {
      setInviteLoading(false);
    }
  };

  const toggleUserSelection = (userId: number) => {
    const newSelection = new Set(selectedUsers);
    if (newSelection.has(userId)) {
      newSelection.delete(userId);
    } else {
      newSelection.add(userId);
    }
    setSelectedUsers(newSelection);
  };

  const handleCreateEvent = async () => {
    if (!newEventTitle.trim() || !newEventTime) return;

    try {
      const formattedTime = newEventTime.replace("T", " ") + ":00";

      await api.post("/events/create", {
        group_id: parseInt(groupId),
        title: newEventTitle,
        description: newEventDescription,
        event_time: formattedTime,
      });

      setNewEventTitle("");
      setNewEventDescription("");
      setNewEventTime("");
      setShowCreateEvent(false);

      // Reload events and their details
      const eventsRes = await api.get(`/group/${groupId}/events`);
      setEvents(eventsRes.data);
      await loadEventDetails(eventsRes.data);
    } catch (err) {
      console.error("Error creating event:", err);
      alert("Failed to create event");
    }
  };

  const handleEventResponse = async (eventId: number, response: "going" | "not_going") => {
    try {
      await api.post("/events/respond", { event_id: eventId, response });
      
      // Optimistically update the user's response
      setEvents(events.map(e => e.event_id === eventId ? { ...e, user_response: response } : e));
      
      // Reload event details to get updated counts
      const eventRes = await api.get(`/event/${eventId}`);
      setEventDetails(prev => ({
        ...prev,
        [eventId]: eventRes.data
      }));
    } catch (err) {
      console.error("Error responding to event:", err);
      alert("Failed to respond to event");
    }
  };

  const handlePostCreated = () => {
    // Reload posts after creation
    loadGroupData();
  };

  const goToChat = () => router.push(`/chat/group/${groupId}`);
  const goToMembers = () => router.push(`/chat/group/${groupId}/members`);
  const goToPost = (postId: number) => router.push(`/post/${postId}`);
  const formatEventTime = (t: string) => new Date(t).toLocaleString();
  const isEventPast = (t: string) => new Date(t) < new Date();

  const canCreateEvent = ["creator", "admin", "member"].includes(group?.user_role || "");
  const canCreatePost = ["creator", "admin", "member"].includes(group?.user_role || "");
  const canInviteUsers = ["creator", "admin"].includes(group?.user_role || "");

  if (loading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 p-4">
        <div className="max-w-6xl mx-auto text-center py-8">
          <div className="text-lg text-gray-600">Loading group...</div>
        </div>
      </div>
    );
  }

  if (!group) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 p-4">
        <div className="max-w-6xl mx-auto text-center py-8">
          <div className="text-lg text-gray-600">Group not found</div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 p-4">
      <div className="max-w-6xl mx-auto space-y-6">
        {/* Header */}
        <div className="bg-white rounded-xl shadow-lg p-6 border border-blue-200">
          <div className="flex justify-between items-start mb-4">
            <div className="flex-1 min-w-0">
              <h1 className="text-3xl font-bold text-gray-900 break-words">{group.title}</h1>
              {group.description && (
                <p className="text-gray-600 mt-2 break-words">{group.description}</p>
              )}
              <div className="flex flex-wrap gap-4 mt-3 text-sm text-gray-500">
                <span className="bg-blue-100 px-2 py-1 rounded-full">{group.member_count} members</span>
                <span className="bg-green-100 px-2 py-1 rounded-full">Created by {group.creator_name}</span>
                <span className="bg-purple-100 px-2 py-1 rounded-full">{new Date(group.created_at).toLocaleDateString()}</span>
              </div>
            </div>
            <div className="flex gap-2">
              <button 
                onClick={goToChat} 
                className="flex items-center gap-2 px-6 py-3 bg-gradient-to-r from-blue-600 to-blue-700 hover:from-blue-700 hover:to-blue-800 text-white rounded-lg shadow-md transition-all duration-200 transform hover:scale-105"
              >
                <MessageCircle className="w-5 h-5" /> Chat
              </button>
              {canInviteUsers && (
                <button 
                  onClick={() => setShowInviteUsers(true)} 
                  className="flex items-center gap-2 px-6 py-3 bg-gradient-to-r from-green-600 to-green-700 hover:from-green-700 hover:to-green-800 text-white rounded-lg shadow-md transition-all duration-200 transform hover:scale-105"
                >
                  <UserPlus className="w-5 h-5" /> Invite
                </button>
              )}
            </div>
          </div>
        </div>

        {showSuccessMessage && (
          <div className="bg-green-100 border border-green-400 text-green-700 px-4 py-3 rounded-lg mb-4">
            {successText}
          </div>
        )}

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Members */}
          <div className="lg:col-span-1 space-y-6">
            <div className="bg-white rounded-xl shadow-lg p-6 border border-blue-200">
              <div className="flex justify-between items-center mb-4">
                <h2 className="text-lg font-bold text-gray-800 flex items-center gap-2">
                  <Users className="w-5 h-5 text-blue-600" /> Members ({group.member_count})
                </h2>
                <button 
                  onClick={goToMembers} 
                  className="text-blue-600 hover:text-blue-800 flex items-center gap-1 transition-colors"
                >
                  See all <ChevronRight className="w-4 h-4" />
                </button>
              </div>
              <div className="space-y-3">
                {!members || members.length === 0 ? (
                  <p className="text-gray-500 text-center py-4">No members found</p>
                ) : (
                  members.slice(0, 5).map(m => (
                    <div key={m.user_id} className="flex items-center gap-3 p-2 rounded-lg hover:bg-blue-50 transition-colors">
                      {m.avatar ? (
                        <img 
                          src={`${process.env.NEXT_PUBLIC_API_BASE_URL}/${m.avatar}`} 
                          alt={m.user_name} 
                          className="w-10 h-10 rounded-full object-cover border-2 border-blue-200"
                        />
                      ) : (
                        <div className="w-10 h-10 rounded-full bg-gradient-to-br from-blue-400 to-blue-600 flex items-center justify-center text-white font-bold text-sm">
                          {m.user_name.charAt(0).toUpperCase()}
                        </div>
                      )}
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="text-sm font-medium truncate text-gray-800">{m.user_name}</span>
                          {onlineUsers.includes(m.user_id) && (
                            <div className="w-3 h-3 bg-green-500 rounded-full border-2 border-white shadow-sm"></div>
                          )}
                        </div>
                        {m.role !== "member" && (
                          <span className="text-xs bg-blue-100 text-blue-700 px-2 py-1 rounded-full capitalize">
                            {m.role}
                          </span>
                        )}
                      </div>
                    </div>
                  ))
                )}
                {members && members.length > 5 && (
                  <div className="text-center text-sm text-gray-500 pt-2 bg-blue-50 rounded-lg p-2">
                    +{members.length - 5} more members
                  </div>
                )}
              </div>
            </div>

            {/* Events */}
            <div className="bg-white rounded-xl shadow-lg p-6 border border-blue-200">
              <div className="flex justify-between items-center mb-4">
                <h2 className="text-lg font-bold text-gray-800 flex items-center gap-2">
                  <Calendar className="w-5 h-5 text-purple-600" /> Events
                </h2>
                {canCreateEvent && (
                  <button 
                    onClick={() => setShowCreateEvent(true)} 
                    className="text-purple-600 hover:text-purple-800 bg-purple-100 hover:bg-purple-200 p-2 rounded-lg transition-colors"
                  >
                    <Plus className="w-5 h-5" />
                  </button>
                )}
              </div>
              <div className="space-y-4">
                {!events || events.length === 0 ? (
                  <p className="text-gray-500 text-center py-4">No events yet</p>
                ) : (
                  events.map(e => {
                    const eventData = eventDetails[e.event_id];
                    const isPast = isEventPast(e.event_time);
                    const goingCount = eventData?.response_counts?.going || 0;
                    const notGoingCount = eventData?.response_counts?.not_going || 0;
                    const totalResponses = goingCount + notGoingCount;
                    const goingPercentage = totalResponses > 0 ? (goingCount / totalResponses) * 100 : 0;
                    
                    return (
                      <div 
                        key={e.event_id} 
                        className={`p-4 border-2 rounded-xl transition-all ${
                          isPast 
                            ? "bg-gray-50 border-gray-200" 
                            : "bg-gradient-to-br from-purple-50 to-blue-50 border-purple-200"
                        }`}
                      >
                        <h3 className="font-bold text-gray-900 mb-2 break-words">{e.title}</h3>
                        {e.description && (
                          <p className="text-sm text-gray-600 mb-3 break-words">{e.description}</p>
                        )}
                        <div className="flex items-center gap-2 text-sm text-gray-500 mb-3">
                          <Clock className="w-4 h-4" />
                          <span className="break-words">{formatEventTime(e.event_time)}</span>
                        </div>

                        {/* Vote Results Visualization */}
                        {totalResponses > 0 && (
                          <div className="mb-3">
                            <div className="flex justify-between text-xs text-gray-600 mb-1">
                              <span>{goingCount} going</span>
                              <span>{notGoingCount} can't go</span>
                            </div>
                            <div className="w-full bg-red-200 rounded-full h-2">
                              <div 
                                className="bg-green-500 h-2 rounded-full transition-all duration-300"
                                style={{ width: `${goingPercentage}%` }}
                              ></div>
                            </div>
                            <div className="text-xs text-center text-gray-500 mt-1">
                              {totalResponses} response{totalResponses !== 1 ? 's' : ''}
                            </div>
                          </div>
                        )}

                        {!isPast && (
                          <div className="flex gap-2">
                            <button 
                              onClick={() => handleEventResponse(e.event_id, "going")} 
                              className={`flex-1 py-2 px-3 rounded-lg text-sm font-medium transition-all transform hover:scale-105 ${
                                e.user_response === "going" 
                                  ? "bg-gradient-to-r from-green-500 to-green-600 text-white shadow-md" 
                                  : "bg-green-100 text-green-700 hover:bg-green-200"
                              }`}
                            >
                              <ThumbsUp className="w-4 h-4 inline mr-1" /> Going
                            </button>
                            <button 
                              onClick={() => handleEventResponse(e.event_id, "not_going")} 
                              className={`flex-1 py-2 px-3 rounded-lg text-sm font-medium transition-all transform hover:scale-105 ${
                                e.user_response === "not_going" 
                                  ? "bg-gradient-to-r from-red-500 to-red-600 text-white shadow-md" 
                                  : "bg-red-100 text-red-700 hover:bg-red-200"
                              }`}
                            >
                              <ThumbsDown className="w-4 h-4 inline mr-1" /> Can't go
                            </button>
                          </div>
                        )}
                      </div>
                    );
                  })
                )}
              </div>
            </div>
          </div>

          {/* Posts */}
          <div className="lg:col-span-2 space-y-6">
            {canCreatePost && (
              <div className="bg-white rounded-xl shadow-lg p-6 border border-blue-200">
                <div className="flex justify-between items-center">
                  <h2 className="text-lg font-bold text-gray-800">Share with the group</h2>
                  <button 
                    onClick={() => setShowCreatePost(true)} 
                    className="flex items-center gap-2 px-6 py-3 bg-gradient-to-r from-green-500 to-green-600 hover:from-green-600 hover:to-green-700 text-white rounded-lg shadow-md transition-all duration-200 transform hover:scale-105"
                  >
                    <Plus className="w-4 h-4" /> New Post
                  </button>
                </div>
              </div>
            )}

            <div className="space-y-4">
              {!posts || posts.length === 0 ? (
                <div className="bg-white rounded-xl shadow-lg p-8 text-center border border-blue-200">
                  <p className="text-gray-500">No posts yet. Be the first to share something!</p>
                </div>
              ) : (
                posts.map(p => (
                  <div key={p.post_id} className="bg-white rounded-xl shadow-lg p-6 border border-blue-200 hover:shadow-xl transition-shadow">
                    <div className="flex items-start gap-4">
                      <div className="w-12 h-12 rounded-full bg-gradient-to-br from-blue-400 to-blue-600 flex items-center justify-center text-white font-bold flex-shrink-0">
                        {p.nickname?.charAt(0).toUpperCase() || "U"}
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-2 flex-wrap">
                          <span className="font-bold text-gray-800">{p.nickname}</span>
                          <span className="text-sm text-gray-500 bg-gray-100 px-2 py-1 rounded-full">
                            {new Date(p.created_at).toLocaleDateString()}
                          </span>
                        </div>
                        <p className="text-gray-900 mb-3 break-words whitespace-pre-wrap">{p.content}</p>
                        {p.media && (
                          <img 
                            src={`${process.env.NEXT_PUBLIC_API_BASE_URL}/${p.media}`} 
                            alt="Post media" 
                            className="max-w-full h-auto rounded-lg mb-3 shadow-md"
                          />
                        )}
                        <button
                          onClick={() => goToPost(p.post_id)}
                          className="flex items-center gap-1 text-sm text-blue-600 hover:text-blue-800 font-medium transition-colors"
                        >
                          <ExternalLink className="w-4 h-4" />
                          View post
                        </button>
                      </div>
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>
        </div>

        {/* Create Event Modal */}
{showCreateEvent && (
  <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
    <div className="bg-gray-50 rounded-2xl shadow-2xl p-6 w-full max-w-md border border-gray-200">
      {/* Title */}
      <h2 className="text-2xl font-bold mb-6 text-gray-900">
        Create Event
      </h2>

      {/* Inputs */}
      <div className="space-y-4">
        <input
          type="text"
          value={newEventTitle}
          onChange={e => setNewEventTitle(e.target.value)}
          placeholder="Event title"
          className="w-full p-4 border border-gray-300 rounded-lg text-black placeholder-gray-400 focus:border-purple-500 focus:ring-2 focus:ring-purple-200 focus:outline-none transition"
        />

        <textarea
          value={newEventDescription}
          onChange={e => setNewEventDescription(e.target.value)}
          placeholder="Event description (optional)"
          className="w-full p-4 border border-gray-300 rounded-lg resize-none text-black placeholder-gray-400 focus:border-purple-500 focus:ring-2 focus:ring-purple-200 focus:outline-none transition"
          rows={3}
        />

        <input
          type="datetime-local"
          value={newEventTime}
          onChange={e => setNewEventTime(e.target.value)}
          className="w-full p-4 border border-gray-300 rounded-lg text-black focus:border-purple-500 focus:ring-2 focus:ring-purple-200 focus:outline-none transition"
        />

        {/* Action buttons */}
        <div className="flex justify-end gap-3 pt-4">
          <button
            onClick={() => {
              setShowCreateEvent(false);
              setNewEventTitle("");
              setNewEventDescription("");
              setNewEventTime("");
            }}
            className="px-6 py-3 text-gray-700 border border-gray-300 rounded-lg hover:bg-gray-100 transition"
          >
            Cancel
          </button>

          <button
            onClick={handleCreateEvent}
            disabled={!newEventTitle.trim() || !newEventTime}
            className="px-6 py-3 bg-gradient-to-r from-purple-500 to-purple-600 text-white rounded-lg shadow-md hover:from-purple-600 hover:to-purple-700 disabled:opacity-50 disabled:cursor-not-allowed transition transform hover:scale-105"
          >
            Create Event
          </button>
        </div>
      </div>
    </div>
  </div>
)}


        {/* Create Post Modal */}
        {showCreatePost && (
          <CreatePostModal
            onClose={() => setShowCreatePost(false)}
            onPostCreated={handlePostCreated}
          />
        )}

        {/* Invite Users Modal */}
        {showInviteUsers && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
            <div className="bg-white rounded-xl shadow-2xl w-full max-w-2xl max-h-[80vh] flex flex-col">
              {/* Header */}
              <div className="p-6 border-b border-gray-200">
                <h2 className="text-2xl font-bold text-gray-800 mb-2">Invite Users to Group</h2>
                <div className="relative">
                  <Search className="w-5 h-5 absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400" />
                  <input
                    type="text"
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    placeholder="Search users by name or email..."
                    className="w-full pl-10 pr-4 py-3 border-2 border-gray-200 rounded-lg focus:border-blue-500 focus:outline-none transition-colors"
                  />
                </div>
                {selectedUsers.size > 0 && (
                  <div className="mt-3 text-sm text-blue-600">
                    {selectedUsers.size} user{selectedUsers.size > 1 ? 's' : ''} selected
                  </div>
                )}
              </div>

              {/* Users List */}
              <div className="flex-1 overflow-y-auto p-6">
                {filteredUsers.length === 0 ? (
                  <div className="text-center py-8 text-gray-500">
                    {searchQuery ? "No users found matching your search" : "All users are already members or no users available"}
                  </div>
                ) : (
                  <div className="space-y-3">
                    {filteredUsers.map((user) => (
                      <div
                        key={user.id}
                        onClick={() => toggleUserSelection(user.id)}
                        className={`flex items-center gap-3 p-3 rounded-lg cursor-pointer transition-colors ${
                          selectedUsers.has(user.id)
                            ? "bg-blue-100 border-2 border-blue-300"
                            : "bg-gray-50 hover:bg-gray-100 border-2 border-transparent"
                        }`}
                      >
                        {/* Checkbox */}
                        <div
                          className={`w-5 h-5 rounded border-2 flex items-center justify-center ${
                            selectedUsers.has(user.id)
                              ? "bg-blue-600 border-blue-600"
                              : "border-gray-300"
                          }`}
                        >
                          {selectedUsers.has(user.id) && (
                            <svg className="w-3 h-3 text-white" fill="currentColor" viewBox="0 0 20 20">
                              <path
                                fillRule="evenodd"
                                d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                                clipRule="evenodd"
                              />
                            </svg>
                          )}
                        </div>

                        {/* User Avatar */}
                        {user.avatar ? (
                          <img
                            src={`${process.env.NEXT_PUBLIC_API_BASE_URL}/${user.avatar}`}
                            alt={user.nickname}
                            className="w-10 h-10 rounded-full object-cover border-2 border-gray-200"
                          />
                        ) : (
                          <div className="w-10 h-10 rounded-full bg-gradient-to-br from-purple-400 to-purple-600 flex items-center justify-center text-white font-bold text-sm">
                            {user.nickname.charAt(0).toUpperCase()}
                          </div>
                        )}

                        {/* User Info */}
                        <div className="flex-1 min-w-0">
                          <div className="font-medium text-gray-800 truncate">{user.nickname}</div>
                          <div className="text-sm text-gray-500 truncate">{user.email}</div>
                        </div>

                        {/* Online Status */}
                        {onlineUsers.includes(user.id) && (
                          <div className="w-3 h-3 bg-green-500 rounded-full border-2 border-white shadow-sm"></div>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* Footer */}
              <div className="p-6 border-t border-gray-200">
                <div className="flex justify-end gap-3">
                  <button
                    onClick={() => {
                      setShowInviteUsers(false);
                      setSelectedUsers(new Set());
                      setSearchQuery("");
                    }}
                    className="px-6 py-3 text-gray-600 border-2 border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
                    disabled={inviteLoading}
                  >
                    Cancel
                  </button>
                  <button
                    onClick={handleInviteUsers}
                    disabled={selectedUsers.size === 0 || inviteLoading}
                    className="px-6 py-3 bg-gradient-to-r from-green-500 to-green-600 text-white rounded-lg hover:from-green-600 hover:to-green-700 disabled:opacity-50 disabled:cursor-not-allowed transition-all transform hover:scale-105"
                  >
                    {inviteLoading 
                      ? "Sending invites..." 
                      : `Invite ${selectedUsers.size} user${selectedUsers.size !== 1 ? 's' : ''}`
                    }
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}