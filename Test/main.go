// Simple Terminal Client to Test APIs

package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/websocket"
)

func main() {
	// Prompt user for their ID
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your user ID: ")
	userID, _ := reader.ReadString('\n')
	userID = strings.TrimSpace(userID) // Remove newline and extra spaces

	// Connect to WebSocket
	url := fmt.Sprintf("ws://localhost:8088/ws?userId=%s", userID)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatal("WebSocket connection failed:", err)
	}
	defer conn.Close()

	log.Println("Connected as user ID:", userID)

	// Listen for messages
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("Read error:", err)
				return
			}
			log.Println("Received message:", string(message))
		}
	}()

	// Allow user to send messages
	for {
		fmt.Print("Enter receiver ID: ")
		receiverID, _ := reader.ReadString('\n')
		receiverID = strings.TrimSpace(receiverID)

		fmt.Print("Enter message: ")
		message, _ := reader.ReadString('\n')
		message = strings.TrimSpace(message)

		msg := fmt.Sprintf(`{"sender_id": "%s", "receiver_id": "%s", "content": "%s"}`, userID, receiverID, message)
		err := conn.WriteMessage(websocket.TextMessage, []byte(msg))
		if err != nil {
			log.Println("Send error:", err)
		} else {
			log.Println("Sent message to user ID", receiverID, ":", message)
			log.Println("Self message confirmation:", message)
		}
	}
}

func testGetUsers() {
	resp, err := http.Get("http://localhost:8088/users")
	if err != nil {
		log.Println("Error fetching users:", err)
		return
	}
	defer resp.Body.Close()

	log.Println("Users response status:", resp.Status)
}

func testGetMessages() {
	resp, err := http.Get("http://localhost:8088/messages?sender_id=1&receiver_id=2")
	if err != nil {
		log.Println("Error fetching messages:", err)
		return
	}
	defer resp.Body.Close()

	log.Println("Messages response status:", resp.Status)
}

// Instructions to run multiple instances:
// 1. Open multiple terminal windows.
// 2. Run `go run main.go` in each terminal.
// 3. Enter a different user ID in each terminal.
// 4. Send messages and observe real-time updates, including self-confirmation.
// 5. Both the sender and receiver should see the messages in real-time.
