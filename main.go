package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	// Initialize the database
	initDB()
	defer db.Close()

	// Serve static files (images/videos)
    fs := http.FileServer(http.Dir("uploads"))
    http.Handle("/uploads/", http.StripPrefix("/uploads/", fs))

	// Set up routes with CORS disabled
	http.HandleFunc("/register", disableCORS(registerHandler))
	http.HandleFunc("/login", disableCORS(loginHandler))

	http.HandleFunc("/posts", disableCORS(jwtMiddleware(createPostHandler)))
	http.HandleFunc("/posts/all", disableCORS(getPostsHandler))
	http.HandleFunc("/post/", disableCORS(GetPostByIDHandler)) 
	http.HandleFunc("/comments", disableCORS(jwtMiddleware(createCommentHandler)))
	http.HandleFunc("/comments/all", disableCORS(getCommentsByPostHandler))


	http.HandleFunc("/ws", disableCORS(handleConnections))
	http.HandleFunc("/messages", disableCORS(getMessagesHandler))
	http.HandleFunc("/online", disableCORS(getOnlineUsers))

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
