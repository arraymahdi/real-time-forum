document.addEventListener("DOMContentLoaded", () => {
    const token = localStorage.getItem("token");
    if (!token) {
        alert("You need to log in first!");
        window.location.href = "login.html";
        return;
    }

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

    async function loadMessages(receiverID) {
        selectedReceiverID = receiverID;
        document.getElementById("messages-list").innerHTML = "";

        try {
            const response = await fetch(`http://localhost:8088/messages?user1=${currentUserID}&user2=${receiverID}`);
            const messages = await response.json();

            messages.forEach(msg => displayMessage(msg, msg.sender_id === currentUserID));
        } catch (error) {
            console.error("Failed to load messages:", error);
        }
    }

    function displayMessage(msg, isSender) {
        const messageList = document.getElementById("messages-list");
        const msgDiv = document.createElement("div");
        msgDiv.classList.add("message", isSender ? "sent" : "received");
        msgDiv.textContent = msg.content;
        messageList.appendChild(msgDiv);
        messageList.scrollTop = messageList.scrollHeight; // Auto-scroll
    }

    // 🔥 Fix: Make sure event listeners are properly attached
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

            console.log("Sending message:", msgData);
            socket.send(JSON.stringify(msgData));
            displayMessage(msgData, true);
            messageInput.value = "";
        } else {
            console.error("No receiver selected or message empty");
        }
    }
});

document.addEventListener("DOMContentLoaded", function () {
    const logoutBtn = document.getElementById("logout-btn");
    if (logoutBtn) {
        logoutBtn.addEventListener("click", logout);
    }
});

function logout() {
    localStorage.removeItem("token"); // Remove auth token
    window.location.href = "login.html"; // Redirect to login page
}

