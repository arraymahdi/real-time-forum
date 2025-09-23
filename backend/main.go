package main

import (
	"backend/pkg/db/sqlite"
	"database/sql"
	"fmt"
	"log"
	"net/http"
)

var db *sql.DB

func main() {
	// Initialize the database
	initDB()
	sqlite.ApplyMigrations()
	defer db.Close()

	// Serve static files (images/videos)
	fs := http.FileServer(http.Dir("uploads"))
	http.Handle("/uploads/", http.StripPrefix("/uploads/", fs))

	// Auth
	http.HandleFunc("/register", withCORS(registerHandler))
	http.HandleFunc("/login", withCORS(loginHandler))
	http.HandleFunc("/upload-avatar", withCORS(uploadAvatarHandler))

	// Posts & comments
	http.HandleFunc("/posts", withCORS(jwtMiddleware(createPostHandler)))
	http.HandleFunc("/posts/all", withCORS(jwtMiddleware(getPostsHandler)))
	http.HandleFunc("/post/", withCORS(jwtMiddleware(GetPostByIDHandler)))
	http.HandleFunc("/comments", withCORS(jwtMiddleware(createCommentHandler)))
	http.HandleFunc("/comments/all", withCORS(getCommentsByPostHandler))

	// Chat & WebSocket
	http.HandleFunc("/ws", withCORS(handleConnections))
	http.HandleFunc("/private-messages", withCORS(jwtMiddleware(getPrivateMessagesHandler)))
	http.HandleFunc("/group-messages", withCORS(jwtMiddleware(getGroupMessagesHandler)))
	http.HandleFunc("/chat-list", withCORS(jwtMiddleware(getMessageableUsersAndGroupsHandler)))

	// Social
	http.HandleFunc("/follow", withCORS(jwtMiddleware(followUserHandler)))
	http.HandleFunc("/unfollow", withCORS(jwtMiddleware(unfollowUserHandler)))
	http.HandleFunc("/followers", withCORS(jwtMiddleware(getFollowersHandler)))
	http.HandleFunc("/following", withCORS(jwtMiddleware(getFollowingHandler)))
	http.HandleFunc("/user/follow-status", withCORS(jwtMiddleware(getUserFollowStatusHandler)))

	// Follow request routes
	http.HandleFunc("/api/follow-requests", jwtMiddleware(getFollowRequestsHandler))
	http.HandleFunc("/api/follow-requests/handle", jwtMiddleware(handleFollowRequestHandler))

	// Groups
	http.HandleFunc("/groups/create", withCORS(jwtMiddleware(createGroupHandler)))
	http.HandleFunc("/groups/browse", withCORS(jwtMiddleware(browseGroupsHandler)))
	http.HandleFunc("/groups/my", withCORS(jwtMiddleware(getUserGroupsHandler)))
	http.HandleFunc("/groups/update", withCORS(jwtMiddleware(updateGroupHandler)))
	http.HandleFunc("/groups/invite", withCORS(jwtMiddleware(inviteToGroupHandler)))
	http.HandleFunc("/groups/request", withCORS(jwtMiddleware(requestJoinGroupHandler)))
	http.HandleFunc("/groups/respond", withCORS(jwtMiddleware(respondToGroupRequestHandler)))
	http.HandleFunc("/groups/pending", withCORS(jwtMiddleware(getPendingRequestsHandler)))
	http.HandleFunc("/groups/leave", withCORS(jwtMiddleware(leaveGroupHandler)))
	http.HandleFunc("/groups/remove", withCORS(jwtMiddleware(removeMemberHandler)))
	http.HandleFunc("/groups/promote", withCORS(jwtMiddleware(promoteMemberHandler)))

	// Events
	http.HandleFunc("/events/create", withCORS(jwtMiddleware(createEventHandler)))
	http.HandleFunc("/events/respond", withCORS(jwtMiddleware(respondToEventHandler)))

	// Dynamic routes
	http.HandleFunc("/group/", withCORS(handleGroupDynamicRoutes))
	http.HandleFunc("/event/", withCORS(handleEventDynamicRoutes))

	//Notifications
	http.HandleFunc("/notifications", withCORS(jwtMiddleware(getNotificationsHandler)))
	http.HandleFunc("/notifications/read", withCORS(jwtMiddleware(markNotificationReadHandler)))
	http.HandleFunc("/notifications/read-all", withCORS(jwtMiddleware(markAllNotificationsReadHandler)))
	http.HandleFunc("/notifications/count", withCORS(jwtMiddleware(getUnreadNotificationCountHandler)))
	http.HandleFunc("/notifications/delete", withCORS(jwtMiddleware(deleteNotificationHandler)))

	// Users
	http.HandleFunc("/users", withCORS(getAllUsersHandler))
	http.HandleFunc("/user/profile", withCORS(jwtMiddleware(getCurrentUserProfileHandler)))
	http.HandleFunc("/user/", withCORS(jwtMiddleware(getUserByIDHandler)))
	http.HandleFunc("/user/profile/details", withCORS(jwtMiddleware(getFullUserProfileHandler)))
	http.HandleFunc("/user/profile/update", withCORS(jwtMiddleware(updateUserProfileHandler)))

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
