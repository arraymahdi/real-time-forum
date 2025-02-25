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

	// Set up routes
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/login", loginHandler)

	http.HandleFunc("/posts", jwtMiddleware(createPostHandler))
	http.HandleFunc("/posts/all", getPostsHandler)
	http.HandleFunc("/comments", jwtMiddleware(createCommentHandler))
	http.HandleFunc("/comments/all", getCommentsHandler)


	http.HandleFunc("/ws", handleConnections)
	http.HandleFunc("/messages", getMessagesHandler)
	http.HandleFunc("/online", getOnlineUsers)


	fmt.Println("Server running on port 8088")
	log.Fatal(http.ListenAndServe(":8088", nil))
}
