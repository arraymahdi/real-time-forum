package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gorilla/websocket"
)

type Message struct {
	SenderID   int    `json:"sender_id"`
	ReceiverID int    `json:"receiver_id"`
	Content    string `json:"content"`
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	// Get User ID
	fmt.Print("Enter your user ID: ")
	userIDStr, _ := reader.ReadString('\n')
	userIDStr = strings.TrimSpace(userIDStr)

	// Connect to WebSocket
	url := "ws://localhost:8088/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatal("WebSocket connection failed:", err)
	}
	defer conn.Close()

	// Send User ID as JSON
	var userID int
	_, err = fmt.Sscanf(userIDStr, "%d", &userID)
	if err != nil {
		log.Fatal("Invalid user ID:", err)
	}

	err = conn.WriteJSON(map[string]int{"user_id": userID})
	if err != nil {
		log.Fatal("Failed to send user ID:", err)
	}

	log.Println("Connected as user ID:", userID)

	// Listen for incoming messages
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("Read error:", err)
				return
			}
			log.Println("Received:", string(message))
		}
	}()

	// Send messages
	for {
		fmt.Print("Enter receiver ID: ")
		receiverIDStr, _ := reader.ReadString('\n')
		receiverIDStr = strings.TrimSpace(receiverIDStr)

		fmt.Print("Enter message: ")
		content, _ := reader.ReadString('\n')
		content = strings.TrimSpace(content)

		var receiverID int
		_, err = fmt.Sscanf(receiverIDStr, "%d", &receiverID)
		if err != nil {
			log.Println("Invalid receiver ID:", err)
			continue
		}

		msg := Message{
			SenderID:   userID,
			ReceiverID: receiverID,
			Content:    content,
		}

		err = conn.WriteJSON(msg)
		if err != nil {
			log.Println("Send error:", err)
		} else {
			log.Println("Sent to user", receiverID, ":", content)
		}
	}
}
