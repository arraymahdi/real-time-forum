
let socket = null;
let currentUserID = null;
let selectedReceiverID = null;

function initializeUsers() {
    const token = localStorage.getItem("token");
    if (!token) {
        console.error("No token found, cannot initialize users");
        return;
    }

    console.log("Initializing WebSocket with token:", token);
    socket = new WebSocket("ws://localhost:8088/ws");

    socket.onopen = () => {
        console.log("WebSocket connected!");
        socket.send(JSON.stringify({ token: token }));
    };

    socket.onmessage = async (event) => {
        const data = JSON.parse(event.data);
        console.log("Received WebSocket message:", data);

        if (data.status === "connected") {
            currentUserID = parseInt(data.user_id);
            console.log("User connected, currentUserID set to:", currentUserID);
            await fetchAndSortUsers();
        } else if (data.type === "online_users") {
            console.log("Updating online users:", data.online_users);
            updateOnlineUsers(data.online_users);
        } else if (data.sender_id && data.receiver_id) {
            if (data.receiver_id === currentUserID || data.sender_id === currentUserID) {
                console.log("New message received:", data);
                displayMessage(data, data.sender_id === currentUserID);
                await fetchAndSortUsers();
            }
        }
    };

    socket.onclose = () => console.log("WebSocket connection closed.");
    socket.onerror = (error) => console.error("WebSocket error:", error);

    // Fallback: If WebSocket fails or no users load after 3 seconds, try a direct fetch
    setTimeout(async () => {
        if (!document.querySelector(".user-item") && currentUserID) {
            console.log("No users loaded via WebSocket, attempting direct fetch...");
            await fetchAndSortUsers();
        } else if (!currentUserID) {
            console.error("currentUserID not set after timeout, WebSocket may have failed");
        }
    }, 3000);
}

async function fetchAndSortUsers() {
    if (!currentUserID) {
        console.error("Cannot fetch users, currentUserID not set");
        return;
    }

    console.log("Fetching users for user_id:", currentUserID);
    try {
        const [usersResponse, onlineResponse] = await Promise.all([
            fetch(`${API_BASE_URL}/getSortedUsers?user_id=${currentUserID}`, {
                headers: { "Authorization": `Bearer ${localStorage.getItem("token")}` }
            }),
            fetch(`${API_BASE_URL}/online`, {
                headers: { "Authorization": `Bearer ${localStorage.getItem("token")}` }
            }),
        ]);

        if (!usersResponse.ok) throw new Error(`Failed to fetch users: ${usersResponse.status}`);
        if (!onlineResponse.ok) throw new Error(`Failed to fetch online users: ${onlineResponse.status}`);

        const users = await usersResponse.json() || [];
        const onlineUsers = await onlineResponse.json() || [];
        console.log("Fetched users:", users);
        console.log("Fetched online users:", onlineUsers);

        const latestMessagesMap = await getLatestMessageTimestamps(users);
        console.log("Latest messages map:", latestMessagesMap);

        const sortedUsers = users.sort((a, b) => {
            const timeA = latestMessagesMap[a.id] || 0;
            const timeB = latestMessagesMap[b.id] || 0;
            if (timeA && timeB) return timeB - timeA;
            if (timeA && !timeB) return -1;
            if (!timeA && timeB) return 1;
            return (a.nickname || "").localeCompare(b.nickname || "");
        });

        console.log("Sorted users:", sortedUsers);
        renderUsers(sortedUsers, onlineUsers, latestMessagesMap);
    } catch (error) {
        console.error("Error fetching users:", error);
    }
}

async function getLatestMessageTimestamps(users) {
    const timestamps = {};
    for (const user of users) {
        if (user.id !== currentUserID) {
            try {
                const response = await fetch(`${API_BASE_URL}/messages?user1=${currentUserID}&user2=${user.id}&offset=0`, {
                    headers: { "Authorization": `Bearer ${localStorage.getItem("token")}` }
                });
                if (!response.ok) throw new Error(`Failed to fetch messages for user ${user.id}: ${response.status}`);
                const messages = await response.json() || [];
                timestamps[user.id] = messages.length > 0 ? new Date(messages[0]?.sent_at).getTime() : 0;
            } catch (error) {
                console.error(`Error fetching messages for user ${user.id}:`, error);
            }
        }
    }
    return timestamps;
}

function renderUsers(users, onlineUsers, latestMessagesMap) {
    const forumUsersList = document.getElementById("forum-users-list");
    const messagesUsersList = document.getElementById("messages-users-list");

    console.log("Rendering users, onlineUsers:", onlineUsers);

    [forumUsersList, messagesUsersList].forEach(usersList => {
        if (usersList) {
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
            console.log(`Rendered users in ${usersList.id}`);
        } else {
            console.error(`Users list element not found: ${usersList ? usersList.id : 'null'}`);
        }
    });
}

function updateOnlineUsers(onlineUsers) {
    console.log("Updating online status for users:", onlineUsers);
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

document.addEventListener("DOMContentLoaded", () => {
    console.log("users.js loaded, initializing...");
    initializeUsers();
});