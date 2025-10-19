package chat

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Message struct {
	SenderID   int    `json:"sender_id"`
	ReceiverID int    `json:"receiver_id,omitempty"`
	GroupID    int    `json:"group_id,omitempty"`
	Content    string `json:"content"`
	SentAt     string `json:"sent_at"`
	Type       string `json:"type,omitempty"` // "private", "group", "typing"
	SenderName string `json:"sender_name,omitempty"`
}

type GroupMessage struct {
	MessageID  int    `json:"message_id"`
	GroupID    int    `json:"group_id"`
	SenderID   int    `json:"sender_id"`
	Content    string `json:"content"`
	Media      string `json:"media,omitempty"`
	CreatedAt  string `json:"created_at"`
	SenderName string `json:"sender_name,omitempty"`
}

type Client struct {
	Conn   *websocket.Conn
	ID     int
	Groups []int // Groups the user is a member of
}

var (
	Clients    = make(map[int]*Client)
	ClientsMux sync.Mutex
)
