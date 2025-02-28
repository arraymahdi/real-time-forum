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
        } else if (data.sender_id && data.receiver_id) {
            if (data.receiver_id === currentUserID || data.sender_id === currentUserID) {
                displayMessage(data, data.sender_id === currentUserID);
            }
        }
    };

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

            // Get latest message timestamps for all users
            const latestMessagesMap = await getLatestMessageTimestamps(users);

            // Sort users based on latest message timestamp (most recent first)
            users.sort((a, b) => {
                const timeA = latestMessagesMap[a.id] || 0;
                const timeB = latestMessagesMap[b.id] || 0;
                return timeB - timeA; // Descending order (latest first)
            });

            renderUsers(users, onlineUsers, latestMessagesMap);
        } catch (error) {
            console.error("Failed to fetch users:", error);
        }
    }

    async function getLatestMessageTimestamps(users) {
        const timestamps = {};

        await Promise.all(users.map(async (user) => {
            if (user.id !== currentUserID) {
                try {
                    const response = await fetch(`http://localhost:8088/messages?user1=${currentUserID}&user2=${user.id}&offset=0`);
                    const messages = await response.json();

                    if (messages.length > 0) {
                        timestamps[user.id] = new Date(messages[0].sent_at).getTime(); // Latest message timestamp
                    } else {
                        timestamps[user.id] = 0; // No messages
                    }
                } catch (error) {
                    console.error(`Failed to fetch messages for user ${user.id}:`, error);
                }
            }
        }));

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

                // Display latest message timestamp (if available)
                const lastMessageTime = latestMessagesMap[user.id] ? new Date(latestMessagesMap[user.id]).toLocaleTimeString() : "No messages";

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

    async function loadMessages(receiverID, initialLoad = true) {
        selectedReceiverID = receiverID;
        offset = 0;
        messages = [];

        document.getElementById("messages-list").innerHTML = "";
        await fetchMessages(initialLoad);
    }

    async function fetchMessages(initialLoad = false) {
        if (loading) return;
        loading = true;

        try {
            const response = await fetch(`http://localhost:8088/messages?user1=${currentUserID}&user2=${selectedReceiverID}&offset=${offset}`);
            const newMessages = await response.json();

            if (newMessages.length > 0) {
                messages = [...newMessages.reverse(), ...messages];
                offset += newMessages.length;
                renderMessages(initialLoad);
            }
        } catch (error) {
            console.error("Failed to load messages:", error);
        } finally {
            loading = false;
        }
    }

    function renderMessages(initialLoad = false) {
        const messageList = document.getElementById("messages-list");
        messageList.innerHTML = "";
    
        if (messages.length === 0) {
            messageList.innerHTML = `<p class="no-messages">No messages yet. Say hello! 👋</p>`;
            return;
        }
    
        messages.forEach(msg => displayMessage(msg, msg.sender_id === currentUserID));
    
        if (initialLoad) {
            messageList.scrollTop = messageList.scrollHeight;
        }
    }
    

    function timeAgo(date) {
        const now = new Date();
        const diff = Math.floor((now - date) / 1000); // Difference in seconds
    
        if (diff < 60) return "Just now";
        if (diff < 3600) return `${Math.floor(diff / 60)} mins ago`;
        if (diff < 86400) return `${Math.floor(diff / 3600)} hours ago`;
        
        return date.toLocaleDateString(); // Show full date if older than a day
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
        
        timestamp.title = sentAt.toLocaleString(); // Show full timestamp on hover
        
        msgDiv.appendChild(text);
        msgDiv.appendChild(timestamp);
        messageList.appendChild(msgDiv);
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
            };

            socket.send(JSON.stringify(msgData));
            displayMessage(msgData, true);
            messageInput.value = "";
        } else {
            console.error("No receiver selected or message empty");
        }
    }

    document.addEventListener("DOMContentLoaded", function () {
        document.querySelectorAll("#logout-btn").forEach(button => {
            button.addEventListener("click", logout);
        });
    });

    function logout() {
        localStorage.removeItem("token");
        window.location.href = "test.html";
    }
});
