document.addEventListener("DOMContentLoaded", () => {
    const token = localStorage.getItem("token");
    if (!token) {
        alert("You need to log in first!");
        window.location.href = "login.html";
        return;
    }

    const socket = new WebSocket("ws://localhost:8088/ws");

    socket.onopen = () => {
        console.log("WebSocket connected!");
        // Send token to authenticate
        socket.send(JSON.stringify({ token: token }));
    };

    socket.onmessage = (event) => {
        const data = JSON.parse(event.data);
        console.log("Message from server:", data);
    };

    socket.onclose = () => {
        console.log("WebSocket connection closed.");
    };

    socket.onerror = (error) => {
        console.error("WebSocket error:", error);
    };

    // Sending messages
    window.sendMessage = function () {
        const messageInput = document.getElementById("message-content");
        const message = messageInput.value.trim();

        if (message !== "") {
            const msgData = { content: message };
            socket.send(JSON.stringify(msgData));
            console.log("Message sent:", msgData);
            messageInput.value = "";
        }
    };
});
