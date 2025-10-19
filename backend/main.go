package main

import (
	"backend/chat"
	"backend/comment"
	"backend/db"
	"backend/event"
	"backend/follower"
	"backend/group"
	"backend/notification"
	"backend/post"

	"backend/pkg/db/sqlite"
	"backend/user"
	"fmt"
	"log"
	"net/http"
)

func main() {
	// Initialize the database
	db.InitDB()
	sqlite.ApplyMigrations()
	defer db.Instance.Close()

	// Serve static files (images/videos)
	fs := http.FileServer(http.Dir("uploads"))
	http.Handle("/uploads/", http.StripPrefix("/uploads/", fs))

	// Auth
	http.HandleFunc("/register", withCORS(user.RegisterHandler))
	http.HandleFunc("/login", withCORS(user.LoginHandler))
	http.HandleFunc("/upload-avatar", withCORS(user.UploadAvatarHandler))

	// Posts & comments
	http.HandleFunc("/posts", withCORS(user.JwtMiddleware(post.CreatePostHandler)))
	http.HandleFunc("/posts/all", withCORS(user.JwtMiddleware(post.GetPostsHandler)))
	http.HandleFunc("/post/", withCORS(user.JwtMiddleware(post.GetPostByIDHandler)))
	http.HandleFunc("/posts/mine", withCORS(user.JwtMiddleware(post.GetMyPostsHandler)))
	http.HandleFunc("/comments", withCORS(user.JwtMiddleware(comment.CreateCommentHandler)))
	http.HandleFunc("/comments/all", withCORS(comment.GetCommentsByPostHandler))

	// Chat & WebSocket
	http.HandleFunc("/ws", withCORS(chat.HandleConnections))
	http.HandleFunc("/private-messages", withCORS(user.JwtMiddleware(chat.GetPrivateMessagesHandler)))
	http.HandleFunc("/group-messages", withCORS(user.JwtMiddleware(chat.GetGroupMessagesHandler)))
	http.HandleFunc("/chat-list", withCORS(user.JwtMiddleware(chat.GetMessageableUsersAndGroupsHandler)))

	// Social
	http.HandleFunc("/follow", withCORS(user.JwtMiddleware(follower.FollowUserHandler)))
	http.HandleFunc("/unfollow", withCORS(user.JwtMiddleware(follower.UnfollowUserHandler)))
	http.HandleFunc("/followers", withCORS(user.JwtMiddleware(follower.GetFollowersHandler)))
	http.HandleFunc("/following", withCORS(user.JwtMiddleware(follower.GetFollowingHandler)))
	http.HandleFunc("/user/follow-status", withCORS(user.JwtMiddleware(follower.GetUserFollowStatusHandler)))

	// Follow request routes
	http.HandleFunc("/api/follow-requests", withCORS(user.JwtMiddleware(follower.GetFollowRequestsHandler)))
	http.HandleFunc("/api/follow-requests/handle", withCORS(user.JwtMiddleware(follower.HandleFollowRequestHandler)))
	http.HandleFunc("/api/follow-request-status", withCORS(user.JwtMiddleware(follower.CheckFollowRequestStatusHandler)))

	// Groups
	http.HandleFunc("/groups/create", withCORS(user.JwtMiddleware(group.CreateGroupHandler)))
	http.HandleFunc("/groups/browse", withCORS(user.JwtMiddleware(group.BrowseGroupsHandler)))
	http.HandleFunc("/groups/my", withCORS(user.JwtMiddleware(group.GetUserGroupsHandler)))
	http.HandleFunc("/groups/update", withCORS(user.JwtMiddleware(group.UpdateGroupHandler)))
	http.HandleFunc("/groups/invite", withCORS(user.JwtMiddleware(group.InviteToGroupHandler)))
	http.HandleFunc("/groups/request", withCORS(user.JwtMiddleware(group.RequestJoinGroupHandler)))
	http.HandleFunc("/groups/respond", withCORS(user.JwtMiddleware(group.RespondToGroupRequestHandler)))
	http.HandleFunc("/groups/pending", withCORS(user.JwtMiddleware(group.GetPendingRequestsHandler)))
	http.HandleFunc("/groups/leave", withCORS(user.JwtMiddleware(group.LeaveGroupHandler)))
	http.HandleFunc("/groups/remove", withCORS(user.JwtMiddleware(group.RemoveMemberHandler)))
	http.HandleFunc("/groups/promote", withCORS(user.JwtMiddleware(group.PromoteMemberHandler)))
	http.HandleFunc("/groups/membership-status", withCORS(user.JwtMiddleware(group.GetMembershipStatusesHandler)))

	// Events
	http.HandleFunc("/events/create", withCORS(user.JwtMiddleware(event.CreateEventHandler)))
	http.HandleFunc("/events/respond", withCORS(user.JwtMiddleware(event.RespondToEventHandler)))

	// Dynamic routes
	http.HandleFunc("/group/", withCORS(group.HandleGroupDynamicRoutes))
	http.HandleFunc("/event/", withCORS(event.HandleEventDynamicRoutes))

	//Notifications
	http.HandleFunc("/notifications", withCORS(user.JwtMiddleware(notification.GetNotificationsHandler)))
	http.HandleFunc("/notifications/read", withCORS(user.JwtMiddleware(notification.MarkNotificationReadHandler)))
	http.HandleFunc("/notifications/read-all", withCORS(user.JwtMiddleware(notification.MarkAllNotificationsReadHandler)))
	http.HandleFunc("/notifications/count", withCORS(user.JwtMiddleware(notification.GetUnreadNotificationCountHandler)))
	http.HandleFunc("/notifications/delete", withCORS(user.JwtMiddleware(notification.DeleteNotificationHandler)))

	// Users
	http.HandleFunc("/users", withCORS(user.GetAllUsersHandler))
	http.HandleFunc("/user/profile", withCORS(user.JwtMiddleware(user.GetCurrentUserProfileHandler)))
	http.HandleFunc("/user/", withCORS(user.JwtMiddleware(user.GetUserByIDHandler)))
	http.HandleFunc("/user/profile/details", withCORS(user.JwtMiddleware(user.GetFullUserProfileHandler)))
	http.HandleFunc("/user/profile/update", withCORS(user.JwtMiddleware(user.UpdateUserProfileHandler)))

	fmt.Println("Server running on port 8088")
	log.Fatal(http.ListenAndServe(":8088", nil))
}

// Global CORS wrapper
func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// You can restrict origin if you want, e.g. "http://localhost:3000"
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}
