document.addEventListener("DOMContentLoaded", () => {
    const token = localStorage.getItem("token");
    const authSection = document.getElementById("auth-section");
    const forumSection = document.getElementById("forum-section");

    console.log("Checking auth, token:", token);

    if (token) {
        authSection.style.display = "none";
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
    localStorage.removeItem("token");
    console.log("Logged out, token removed");
    document.getElementById("auth-section").style.display = "block";
    document.getElementById("forum-section").style.display = "none";
    document.getElementById("messages-section").style.display = "none";
}