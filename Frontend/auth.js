document.addEventListener("DOMContentLoaded", function () {
    const authSection = document.getElementById("auth-section");
    const forumSection = document.getElementById("forum-section");
    const logoutBtn = document.getElementById("logout-btn");

    const signupModal = document.getElementById("signup-modal");
    const openSignup = document.getElementById("open-signup");
    const closeSignup = document.getElementById("close-signup");

    // Check if user is logged in
    const token = localStorage.getItem("token");
    if (token) {
        authSection.style.display = "none";
        forumSection.style.display = "block";
    } else {
        authSection.style.display = "block";
        forumSection.style.display = "none";
    }

    // Open signup modal
    if (openSignup && signupModal) {
        openSignup.addEventListener("click", function () {
            signupModal.style.display = "flex";
        });
    }

    // Close signup modal
    if (closeSignup && signupModal) {
        closeSignup.addEventListener("click", function () {
            signupModal.style.display = "none";
        });
    }

    if (signupModal) {
        window.addEventListener("click", function (event) {
            if (event.target === signupModal) {
                signupModal.style.display = "none";
            }
        });
    }

    // Logout function
    if (logoutBtn) {
        logoutBtn.addEventListener("click", function () {
            localStorage.removeItem("token");
            authSection.style.display = "block";
            forumSection.style.display = "none";
        });
    }
});

// Login function
async function login() {
    const usernameField = document.getElementById("login-username");
    const passwordField = document.getElementById("login-password");

    if (!usernameField || !passwordField) return;

    const user = {
        email: usernameField.value.trim(),
        password: passwordField.value.trim(),
    };

    try {
        const response = await fetch("http://localhost:8088/login", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(user),
        });

        if (!response.ok) throw new Error("Login failed");

        const data = await response.json();
        localStorage.setItem("token", data.token);

        // Show forum and hide auth
        document.getElementById("auth-section").style.display = "none";
        document.getElementById("forum-section").style.display = "block";
    } catch (error) {
        alert(error.message);
    }
}

// Register function
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

    try {
        const response = await fetch("http://localhost:8088/register", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(user),
        });

        if (!response.ok) {
            const errorMsg = await response.text();
            throw new Error(`Registration failed: ${errorMsg}`);
        }

        alert("Registration successful! You can now log in.");
        document.getElementById("signup-modal").style.display = "none";
    } catch (error) {
        alert(error.message);
    }
}

function checkAuth() {
    const token = localStorage.getItem("token");

    if (!token) {
        // Show login, hide everything else
        document.getElementById("auth-section").style.display = "block";
        document.getElementById("forum-section").style.display = "none";
        document.getElementById("messages-section").style.display = "none";
    } else {
        // Show forum by default, hide login
        document.getElementById("auth-section").style.display = "none";
        document.getElementById("forum-section").style.display = "block";
        document.getElementById("messages-section").style.display = "none";
    }
}

document.addEventListener("DOMContentLoaded", () => {
    checkAuth();

    const messagesBtn = document.getElementById("messages-btn");
    const forumBtn = document.getElementById("forum-btn");

    if (messagesBtn) {
        messagesBtn.addEventListener("click", () => {
            document.getElementById("forum-section").style.display = "none";
            document.getElementById("messages-section").style.display = "block";
        });
    }

    if (forumBtn) {
        forumBtn.addEventListener("click", () => {
            document.getElementById("messages-section").style.display = "none";
            document.getElementById("forum-section").style.display = "block";
        });
    }
});

document.addEventListener("DOMContentLoaded", () => {
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
});

// Fix logout to stay in test.html
function logout() {
    localStorage.removeItem("token");
    checkAuth(); // Just update UI
}
