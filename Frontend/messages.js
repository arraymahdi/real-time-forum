document.addEventListener("DOMContentLoaded", () => {
    const token = localStorage.getItem("token");

    const socket = new WebSocket("ws://localhost:8088/ws");
    let currentUserID = null;
    let selectedReceiverID = null;

    socket.onopen = () => {
        console.log("WebSocket connected!");
        socket.send(JSON.stringify({ token: token }));
    };

    socket.onmessage = (event) => {
        const data = JSON.parse(event.data);
        console.log("Message from server:", data);

        if (data.status === "connected") {
            currentUserID = parseInt(data.user_id);
            fetchUsers();
        } else if (data.sender_id && data.receiver_id) {
            if (data.receiver_id === currentUserID || data.sender_id === currentUserID) {
                displayMessage(data, data.sender_id === currentUserID);
            }
        }
    };

    socket.onclose = () => console.log("WebSocket connection closed.");
    socket.onerror = (error) => console.error("WebSocket error:", error);

    async function fetchUsers() {
        try {
            const [usersResponse, onlineResponse] = await Promise.all([
                fetch("http://localhost:8088/users"),
                fetch("http://localhost:8088/online"),
            ]);

            const users = await usersResponse.json();
            const onlineUsers = await onlineResponse.json();

            renderUsers(users, onlineUsers);
        } catch (error) {
            console.error("Failed to fetch users:", error);
        }
    }

    function renderUsers(users, onlineUsers) {
        const usersList = document.getElementById("users-list");
        usersList.innerHTML = "";
    
        // Sort users alphabetically by nickname (case-insensitive)
        users.sort((a, b) => a.nickname.toLowerCase().localeCompare(b.nickname.toLowerCase()));
    
        users.forEach(user => {
            if (user.id !== currentUserID) {
                const userItem = document.createElement("div");
                userItem.classList.add("user-item");
                userItem.dataset.id = user.id;
                userItem.innerHTML = `
                    <span class="status-dot ${onlineUsers.includes(user.id) ? 'online' : 'offline'}"></span>
                    ${user.nickname}
                `;
    
                userItem.addEventListener("click", () => {
                    document.getElementById("chat-header").textContent = `Chat with ${user.nickname}`;
                    loadMessages(user.id);
                });
    
                usersList.appendChild(userItem);
            }
        });
    }
    
    let messages = []; // Store all messages
    let offset = 0;
    const limit = 10;
    let loading = false;

    async function loadMessages(receiverID, initialLoad = true) {
        selectedReceiverID = receiverID;
        offset = 0; // Reset offset on new chat
        messages = []; // Clear stored messages

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
                messages = [...newMessages.reverse(), ...messages]; // Add at the beginning
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
        const prevScrollHeight = messageList.scrollHeight;

        messageList.innerHTML = ""; // Clear and re-render

        messages.forEach(msg => displayMessage(msg, msg.sender_id === currentUserID));

        if (initialLoad) {
            messageList.scrollTop = messageList.scrollHeight; // Scroll to bottom on first load
        } else {
            messageList.scrollTop = messageList.scrollHeight - prevScrollHeight; // Maintain position
        }
    }

    function displayMessage(msg, isSender) {
        const messageList = document.getElementById("messages-list");
        const msgDiv = document.createElement("div");
        msgDiv.classList.add("message", isSender ? "sent" : "received");
    
        const text = document.createElement("p");
        text.textContent = msg.content;
    
        const timestamp = document.createElement("span");
        timestamp.classList.add("timestamp");
    
        // Fix invalid date issue by ensuring it's properly formatted
        const sentAt = msg.sent_at ? new Date(msg.sent_at).toLocaleString() : "Sending...";
        timestamp.textContent = sentAt;
    
        msgDiv.appendChild(text);
        msgDiv.appendChild(timestamp);
        messageList.appendChild(msgDiv);
    }
    

    // Scroll event to load more messages
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
});

document.addEventListener("DOMContentLoaded", function () {
    document.querySelectorAll("#logout-btn").forEach(button => {
        button.addEventListener("click", logout);
    });
});

function logout() {
    localStorage.removeItem("token");
    window.location.href = "test.html";
}
