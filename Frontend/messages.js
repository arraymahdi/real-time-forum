// Declare variables in the outer scope so they’re accessible to all functions
let messages = [];
let offset = 0;
let loading = false;
let typingTimeout;

document.addEventListener("DOMContentLoaded", () => {
    const token = localStorage.getItem("token");
    if (!token) {
        window.location.href = "index.html";
        return;
    }

    console.log("Messages.js loaded, waiting for WebSocket from users.js");

    const forumSection = document.getElementById("forum-section");
    const messagesSection = document.getElementById("messages-section");
    const messagesBtn = document.getElementById("messages-btn");
    const explorePostsBtn = document.getElementById("explore-posts-btn");

    messagesBtn.addEventListener("click", () => {
        forumSection.style.display = "none";
        messagesSection.style.display = "block";
    });

    explorePostsBtn.addEventListener("click", () => {
        messagesSection.style.display = "none";
        forumSection.style.display = "block";
    });

    document.getElementById("messages-list").addEventListener("scroll", function () {
        if (this.scrollTop === 0) fetchMessages();
    });

    document.getElementById("send-btn").addEventListener("click", sendMessage);
    
    const messageInput = document.getElementById("message-content");
    messageInput.addEventListener("keypress", (e) => {
        if (e.key === "Enter") sendMessage();
    });

    // Add typing event listener (broadcast to receiver)
    messageInput.addEventListener("input", () => {
        if (!socket || socket.readyState !== WebSocket.OPEN) return;

        clearTimeout(typingTimeout);
        socket.send(JSON.stringify({
            sender_id: currentUserID,
            receiver_id: selectedReceiverID,
            type: "typing"
        }));

        typingTimeout = setTimeout(() => {
            // Could send a "stopped typing" message here if desired
        }, 2000);
    });
});

async function loadMessages(receiverID) {
    if (!currentUserID) {
        console.error("currentUserID not set yet");
        return;
    }
    selectedReceiverID = receiverID;
    offset = 0;
    messages = [];
    const messageList = document.getElementById("messages-list");
    messageList.innerHTML = ""; // Clear existing content, including typing indicators
    console.log(`Loading messages for receiver ID: ${receiverID}`);
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
        if (!response.ok) throw new Error(`Failed to fetch messages: ${response.status}`);
        const newMessages = await response.json();
        console.log("Fetched messages:", newMessages);

        if (newMessages.length > 0) {
            newMessages.reverse();
            messages = [...newMessages.filter(msg => !msg.type || msg.type !== "typing"), ...messages]; // Filter out typing events
            offset += newMessages.length;
            renderMessages();
            messageList.scrollTop = prevScrollTop + (messageList.scrollHeight - prevScrollHeight);
        }
    } catch (error) {
        console.log("Failed to load messages:", error);
    } finally {
        loading = false;
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

    // Typing indicator for receiver only
    if (msg.type === "typing" && !isSender && msg.sender_id === selectedReceiverID) {
        const existingTyping = document.getElementById("typing-indicator");
        if (!existingTyping) {
            const typingDiv = document.createElement("div");
            typingDiv.id = "typing-indicator";
            typingDiv.classList.add("typing-indicator");
            typingDiv.innerHTML = `<span class="typing-wave"></span>`;
            messageList.appendChild(typingDiv);
            messageList.scrollTop = messageList.scrollHeight;

            // Remove typing indicator after 2 seconds
            setTimeout(() => {
                const typingElement = document.getElementById("typing-indicator");
                if (typingElement) typingElement.remove();
            }, 2000);
        }
        return;
    }

    // Regular message (only if not a typing event)
    if (!msg.type || msg.type !== "typing") {
        const msgDiv = document.createElement("div");
        msgDiv.classList.add("message", isSender ? "sent" : "received");
        msgDiv.innerHTML = `
            <p>${msg.content}</p>
            <span class="timestamp">${timeAgo(new Date(msg.sent_at))}</span>
        `;
        messageList.appendChild(msgDiv);
    }
}

function timeAgo(date) {
    const now = new Date();
    const diff = Math.floor((now - date) / 1000);
    if (diff < 60) return "Just now";
    if (diff < 3600) return `${Math.floor(diff / 60)} mins ago`;
    if (diff < 86400) return `${Math.floor(diff / 3600)} hours ago`;
    return date.toLocaleDateString();
}

function sendMessage() {
    const messageInput = document.getElementById("message-content");
    const message = messageInput.value.trim();

    if (!socket || socket.readyState !== WebSocket.OPEN) {
        console.error("WebSocket not connected");
        return;
    }

    if (message && selectedReceiverID) {
        const msgData = {
            sender_id: currentUserID,
            receiver_id: selectedReceiverID,
            content: message,
            sent_at: new Date().toISOString()
        };
        console.log("Sending message:", msgData);
        socket.send(JSON.stringify(msgData));
        displayMessage(msgData, true);
        messageInput.value = "";
    } else {
        console.error("No receiver selected or message empty");
    }
}