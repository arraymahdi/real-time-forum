document.addEventListener("DOMContentLoaded", function () {
    const pages = {
        home: document.getElementById("home-page"),
        posts: document.getElementById("posts-page"),
        messages: document.getElementById("messages-page"),
        auth: document.getElementById("auth-page")
    };

    function navigateTo(page) {
        Object.values(pages).forEach(p => p.style.display = "none");
        if (pages[page]) {
            pages[page].style.display = "block";
            history.pushState({ page }, "", `#${page}`);
        }
    }

    document.querySelectorAll("nav a").forEach(link => {
        link.addEventListener("click", function (event) {
            event.preventDefault();
            const page = this.getAttribute("href").substring(1);
            navigateTo(page);
        });
    });

    window.addEventListener("popstate", function (event) {
        if (event.state && event.state.page) {
            navigateTo(event.state.page);
        }
    });

    const initialPage = location.hash.substring(1) || "home";
    navigateTo(initialPage);

    // Authentication Handling
    function checkAuth() {
        const token = localStorage.getItem("jwt_token");
        if (!token) {
            navigateTo("auth");
        }
    }

    document.getElementById("login-form")?.addEventListener("submit", function (event) {
        event.preventDefault();
        const formData = new FormData(this);
        fetch("/api/login", {
            method: "POST",
            body: formData
        }).then(response => response.json())
          .then(data => {
              if (data.token) {
                  localStorage.setItem("jwt_token", data.token);
                  navigateTo("home");
              }
          });
    });

    document.getElementById("logout-btn")?.addEventListener("click", function () {
        localStorage.removeItem("jwt_token");
        navigateTo("auth");
    });

    // WebSocket Handling for Messages
    let ws;
    function connectWebSocket() {
        ws = new WebSocket("ws://yourserver.com/ws");
        ws.onmessage = function (event) {
            const message = JSON.parse(event.data);
            const messagesContainer = document.getElementById("messages-container");
            const messageElement = document.createElement("div");
            messageElement.textContent = `${message.sender}: ${message.text}`;
            messagesContainer.appendChild(messageElement);
        };
    }

    if (pages.messages) {
        connectWebSocket();
    }

    checkAuth();
});
