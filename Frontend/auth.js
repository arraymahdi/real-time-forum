document.addEventListener("DOMContentLoaded", () => {
    const token = localStorage.getItem("token");
    const authSection = document.getElementById("auth-section");
    const forumSection = document.getElementById("forum-section");

    console.log("Checking auth, token:", token);

    if (token) {
        authSection.style.display = "none";
        initializeUsers();
        forumSection.style.display = "block";
    } else {
        authSection.style.display = "block";
        forumSection.style.display = "none";
    }

    const signupModal = document.getElementById("signup-modal");
    const openSignup = document.getElementById("open-signup");
    const closeSignup = document.getElementById("close-signup");

    if (openSignup) {
        openSignup.addEventListener("click", () => signupModal.style.display = "flex");
    }
    if (closeSignup) {
        closeSignup.addEventListener("click", () => signupModal.style.display = "none");
    }
    if (signupModal) {
        window.addEventListener("click", (event) => {
            if (event.target === signupModal) signupModal.style.display = "none";
        });
    }

    document.querySelectorAll("#logout-btn").forEach(button => {
        button.addEventListener("click", logout);
    });
});

async function login() {
    const usernameField = document.getElementById("login-username");
    const passwordField = document.getElementById("login-password");

    if (!usernameField || !passwordField) return;

    const user = {
        email: usernameField.value.trim(),
        password: passwordField.value.trim(),
    };

    console.log("Attempting login with:", user);

    try {
        const response = await fetch("http://localhost:8088/login", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(user),
        });

        if (!response.ok) throw new Error(`Login failed: ${await response.text()}`);

        const data = await response.json();
        localStorage.setItem("token", data.token);
        console.log("Login successful, token:", data.token);
        document.getElementById("auth-section").style.display = "none";
        document.getElementById("forum-section").style.display = "block";
    } catch (error) {
        console.error("Login error:", error);
        alert(error.message);
    }
    initializeUsers();
}

async function register() {
    const user = {
        nickname: document.getElementById("register-nickname").value.trim(),
        age: parseInt(document.getElementById("register-age").value, 10),
        gender: document.getElementById("register-gender").value,
        first_name: document.getElementById("register-first-name").value.trim(),
        last_name: document.getElementById("register-last-name").value.trim(),
        email: document.getElementById("register-email").value.trim(),
        password: document.getElementById("register-password").value.trim(),
    };

    console.log("Attempting registration with:", user);

    try {
        const response = await fetch("http://localhost:8088/register", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(user),
        });

        if (!response.ok) throw new Error(`Registration failed: ${await response.text()}`);
        console.log("Registration successful");
        alert("Registration successful! You can now log in.");
        document.getElementById("signup-modal").style.display = "none";
    } catch (error) {
        console.error("Registration error:", error);
        alert(error.message);
    }
}

function logout() {
    if (socket) {
        socket.close();
    }
    localStorage.removeItem("token");
    console.log("Logged out, token removed");
    document.getElementById("auth-section").style.display = "block";
    document.getElementById("forum-section").style.display = "none";
    document.getElementById("messages-section").style.display = "none";
}

let socket = null;
let currentUserID = null;
let selectedReceiverID = null;

function initializeUsers() {
    const token = localStorage.getItem("token");
    if (!token) {
        console.log("No token found, cannot initialize users");
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
            console.log("Real-time online users update:", data.online_users);
            updateOnlineUsers(data.online_users);
        } else if (data.sender_id && data.receiver_id) {
            if (data.receiver_id === currentUserID || data.sender_id === currentUserID) {
                console.log("New message received:", data);
                displayMessage(data, data.sender_id === currentUserID);
                if (data.type === "typing" && data.sender_id !== currentUserID) {
                    showTypingInUserList(data.sender_id);
                } else {
                    await fetchAndSortUsers();
                }
            }
        }
    };

    socket.onclose = () => {
        console.log("WebSocket connection closed.");
        currentUserID = null;
    };

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
            fetch(`http://localhost:8088/getSortedUsers?user_id=${currentUserID}`, {
                headers: { "Authorization": `Bearer ${localStorage.getItem("token")}` }
            }),
            fetch(`http://localhost:8088/online`, {
                headers: { "Authorization": `Bearer ${localStorage.getItem("token")}` }
            }),
        ]);

        if (!usersResponse.ok) throw new Error(`Failed to fetch users: ${usersResponse.status}`);
        if (!onlineResponse.ok) throw new Error(`Failed to fetch online users: ${onlineResponse.status}`);

        const users = await usersResponse.json() || [];
        const onlineUsers = await onlineResponse.json() || [];
        console.log("Fetched users:", users);
        console.log("Fetched online users (initial):", onlineUsers);

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
                const response = await fetch(`http://localhost:8088/messages?user1=${currentUserID}&user2=${user.id}&offset=0`, {
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
                        ${user.nickname}
                        <span class="typing-wave" id="typing-wave-${user.id}" style="display: none;"></span>
                        <small class="last-message-time">${lastMessageTime}</small>
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

function showTypingInUserList(senderId) {
    const userItems = document.querySelectorAll(`.user-item[data-id="${senderId}"]`);
    userItems.forEach(item => {
        const typingWave = item.querySelector(`#typing-wave-${senderId}`);
        if (typingWave) typingWave.style.display = "inline-block";
    });
    setTimeout(() => {
        userItems.forEach(item => {
            const typingWave = item.querySelector(`#typing-wave-${senderId}`);
            if (typingWave) typingWave.style.display = "none";
        });
    }, 2000);
}

function updateOnlineUsers(onlineUsers) {
    console.log("Updating online status for users:", onlineUsers);
    document.querySelectorAll(".user-item").forEach(userItem => {
        const userId = parseInt(userItem.dataset.id);
        const statusDot = userItem.querySelector(".status-dot");
        if (statusDot) {
            if (onlineUsers.includes(userId)) {
                statusDot.classList.add("online");
                statusDot.classList.remove("offline");
            } else {
                statusDot.classList.add("offline");
                statusDot.classList.remove("online");
            }
        }
    });
}