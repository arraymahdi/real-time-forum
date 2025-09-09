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

	// Set up routes with CORS disabled
	http.HandleFunc("/register", disableCORS(registerHandler))
	http.HandleFunc("/login", disableCORS(loginHandler))
	http.HandleFunc("/upload-avatar", disableCORS(uploadAvatarHandler))

	http.HandleFunc("/posts", disableCORS(jwtMiddleware(createPostHandler)))
	http.HandleFunc("/posts/all", disableCORS(jwtMiddleware(getPostsHandler)))
	http.HandleFunc("/post/", disableCORS(jwtMiddleware(GetPostByIDHandler)))
	http.HandleFunc("/comments", disableCORS(jwtMiddleware(createCommentHandler)))
	http.HandleFunc("/comments/all", disableCORS(getCommentsByPostHandler))

	http.HandleFunc("/ws", disableCORS(handleConnections))
	http.HandleFunc("/messages", disableCORS(getMessagesHandler))
	http.HandleFunc("/online", disableCORS(getOnlineUsers))
	http.HandleFunc("/getSortedUsers", disableCORS(getSortedUsersHandler))

	http.HandleFunc("/follow", disableCORS(jwtMiddleware(followUserHandler)))
	http.HandleFunc("/unfollow", disableCORS(jwtMiddleware(unfollowUserHandler)))
	http.HandleFunc("/followers", disableCORS(jwtMiddleware(getFollowersHandler)))
	http.HandleFunc("/following", disableCORS(jwtMiddleware(getFollowingHandler)))
	http.HandleFunc("/user/follow-status", disableCORS(jwtMiddleware(getUserFollowStatusHandler)))

	http.HandleFunc("/groups/create", jwtMiddleware(createGroupHandler))
	http.HandleFunc("/groups/browse", jwtMiddleware(browseGroupsHandler))
	http.HandleFunc("/groups/my", jwtMiddleware(getUserGroupsHandler))
	http.HandleFunc("/groups/update", jwtMiddleware(updateGroupHandler))

	http.HandleFunc("/groups/invite", jwtMiddleware(inviteToGroupHandler))
	http.HandleFunc("/groups/request", jwtMiddleware(requestJoinGroupHandler))
	http.HandleFunc("/groups/respond", jwtMiddleware(respondToGroupRequestHandler))
	http.HandleFunc("/groups/pending", jwtMiddleware(getPendingRequestsHandler))
	http.HandleFunc("/groups/leave", jwtMiddleware(leaveGroupHandler))
	http.HandleFunc("/groups/remove", jwtMiddleware(removeMemberHandler))
	http.HandleFunc("/groups/promote", jwtMiddleware(promoteMemberHandler))

	http.HandleFunc("/events/create", jwtMiddleware(createEventHandler))
	http.HandleFunc("/events/respond", jwtMiddleware(respondToEventHandler))

	// Dynamic routes handler
	http.HandleFunc("/group/", handleGroupDynamicRoutes)
	http.HandleFunc("/event/", handleEventDynamicRoutes)

	http.HandleFunc("/users", disableCORS(getAllUsersHandler))

	fmt.Println("Server running on port 8088")
	log.Fatal(http.ListenAndServe(":8088", nil))
}

func disableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}
