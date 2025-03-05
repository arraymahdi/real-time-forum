document.addEventListener("DOMContentLoaded", async () => {
    const token = localStorage.getItem("token");

    const socket = new WebSocket("ws://localhost:8088/ws");
    let currentUserID = null;
    let selectedReceiverID = null;

    socket.onopen = () => {
        console.log("WebSocket connected!");
        socket.send(JSON.stringify({ token: token }));
    };

    socket.onmessage = async (event) => {
        const data = JSON.parse(event.data);
        console.log("Message from server:", data);
    
        if (data.status === "connected") {
            currentUserID = parseInt(data.user_id);
            await fetchAndSortUsers();
        } else if (data.type === "online_users") {
            updateOnlineUsers(data.online_users);
        } else if (data.sender_id && data.receiver_id) {
            if (data.receiver_id === currentUserID || data.sender_id === currentUserID) {
                displayMessage(data, data.sender_id === currentUserID);
                await fetchAndSortUsers(); // Refresh users when a new message arrives
            }
        }
    };

    function updateOnlineUsers(onlineUsers) {
        document.querySelectorAll(".user-item").forEach(userItem => {
            const userId = parseInt(userItem.dataset.id);
            const statusDot = userItem.querySelector(".status-dot");
            if (onlineUsers.includes(userId)) {
                statusDot.classList.add("online");
                statusDot.classList.remove("offline");
            } else {
                statusDot.classList.add("offline");
                statusDot.classList.remove("online");
            }
        });
    }
    
    
    socket.onclose = () => console.log("WebSocket connection closed.");
    socket.onerror = (error) => console.error("WebSocket error:", error);

    async function fetchAndSortUsers() {
        try {
            const [usersResponse, onlineResponse] = await Promise.all([
                fetch("http://localhost:8088/users"),
                fetch("http://localhost:8088/online"),
            ]);

            const users = await usersResponse.json();
            const onlineUsers = await onlineResponse.json();
            const latestMessagesMap = await getLatestMessageTimestamps(users);

            users.sort((a, b) => {
                const timeA = latestMessagesMap[a.id] || 0;
                const timeB = latestMessagesMap[b.id] || 0;
                return timeB - timeA;
            });

            renderUsers(users, onlineUsers, latestMessagesMap);
        } catch (error) {
            console.error("Failed to fetch users:", error);
        }
    }

    async function getLatestMessageTimestamps(users) {
        const timestamps = {};
        for (const user of users) {
            if (user.id !== currentUserID) {
                try {
                    const response = await fetch(`http://localhost:8088/messages?user1=${currentUserID}&user2=${user.id}&offset=0`);
                    const messages = await response.json();

                    timestamps[user.id] = messages.length > 0
                        ? new Date(messages[0].sent_at).getTime()
                        : 0;
                } catch (error) {
                    console.error(`Failed to fetch messages for user ${user.id}:`, error);
                }
            }
        }
        return timestamps;
    }

    function renderUsers(users, onlineUsers, latestMessagesMap) {
        const usersList = document.getElementById("users-list");
        usersList.innerHTML = "";

        users.forEach(user => {
            if (user.id !== currentUserID) {
                const userItem = document.createElement("div");
                userItem.classList.add("user-item");
                userItem.dataset.id = user.id;

                const lastMessageTime = latestMessagesMap[user.id]
                    ? new Date(latestMessagesMap[user.id]).toLocaleTimeString()
                    : "No messages";

                userItem.innerHTML = `
                    <span class="status-dot ${onlineUsers.includes(user.id) ? 'online' : 'offline'}"></span>
                    ${user.nickname} <small class="last-message-time">${lastMessageTime}</small>
                `;

                userItem.addEventListener("click", () => {
                    document.getElementById("chat-header").textContent = `Chat with ${user.nickname}`;
                    loadMessages(user.id);
                });

                usersList.appendChild(userItem);
            }
        });
    }

    let messages = [];
    let offset = 0;
    const limit = 10;
    let loading = false;

    async function loadMessages(receiverID) {
        selectedReceiverID = receiverID;
        offset = 0;
        messages = [];

        document.getElementById("messages-list").innerHTML = "";
        await fetchMessages();
    }

    async function fetchMessages() {
        if (loading) return;
        loading = true;
    
        const messageList = document.getElementById("messages-list");
        const prevScrollHeight = messageList.scrollHeight;
        const prevScrollTop = messageList.scrollTop;
    
        try {
            const response = await fetch(`http://localhost:8088/messages?user1=${currentUserID}&user2=${selectedReceiverID}&offset=${offset}`);
            const newMessages = await response.json();
    
            if (newMessages.length > 0) {
                newMessages.reverse();
                messages = [...newMessages, ...messages];
                offset += newMessages.length;
                renderMessages(false);
            }
        } catch (error) {
            console.error("Failed to load messages:", error);
        } finally {
            loading = false;
    
            // Preserve scroll position after loading older messages
            messageList.scrollTop = prevScrollTop + (messageList.scrollHeight - prevScrollHeight);
        }
    }    

    function renderMessages() {
        const messageList = document.getElementById("messages-list");
        messageList.innerHTML = "";

        if (messages.length === 0) {
            messageList.innerHTML = `<p class="no-messages">No messages yet. Say hello! 👋</p>`;
            return;
        }

        messages.forEach(msg => displayMessage(msg, msg.sender_id === currentUserID));
        messageList.scrollTop = messageList.scrollHeight;
    }

    function displayMessage(msg, isSender) {
        const messageList = document.getElementById("messages-list");
        const msgDiv = document.createElement("div");
        msgDiv.classList.add("message", isSender ? "sent" : "received");

        const text = document.createElement("p");
        text.textContent = msg.content;

        const timestamp = document.createElement("span");
        timestamp.classList.add("timestamp");

        const sentAt = msg.sent_at ? new Date(msg.sent_at) : new Date();
        timestamp.textContent = timeAgo(sentAt);
        timestamp.title = sentAt.toLocaleString();

        msgDiv.appendChild(text);
        msgDiv.appendChild(timestamp);
        messageList.appendChild(msgDiv);
    }

    function timeAgo(date) {
        const now = new Date();
        const diff = Math.floor((now - date) / 1000);

        if (diff < 60) return "Just now";
        if (diff < 3600) return `${Math.floor(diff / 60)} mins ago`;
        if (diff < 86400) return `${Math.floor(diff / 3600)} hours ago`;

        return date.toLocaleDateString();
    }

    document.getElementById("messages-list").addEventListener("scroll", function () {
        if (this.scrollTop === 0) {
            fetchMessages();
        }
    });

    document.addEventListener("click", (event) => {
        if (event.target.id === "send-btn") {
            sendMessage();
        }
    });

    document.getElementById("message-content").addEventListener("keypress", (e) => {
        if (e.key === "Enter") sendMessage();
    });

    function sendMessage() {
        const messageInput = document.getElementById("message-content");
        const message = messageInput.value.trim();

        if (message && selectedReceiverID) {
            const msgData = {
                sender_id: currentUserID,
                receiver_id: selectedReceiverID,
                content: message,
                sent_at: new Date().toISOString()
            };

            socket.send(JSON.stringify(msgData));
            displayMessage(msgData, true);
            messageInput.value = "";
        } else {
            console.error("No receiver selected or message empty");
        }
    }
});
