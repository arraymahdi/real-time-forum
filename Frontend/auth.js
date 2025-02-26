document.addEventListener("DOMContentLoaded", function () {
    const signupModal = document.getElementById("signup-modal");
    const openSignup = document.getElementById("open-signup");
    const closeSignup = document.getElementById("close-signup");

    // Open signup modal
    openSignup.addEventListener("click", function () {
        signupModal.style.display = "flex";
    });

    // Close signup modal
    closeSignup.addEventListener("click", function () {
        signupModal.style.display = "none";
    });

    // Close modal if clicked outside
    window.addEventListener("click", function (event) {
        if (event.target === signupModal) {
            signupModal.style.display = "none";
        }
    });
});

// Login function
async function login() {
    const user = {
        email: document.getElementById("login-username").value,
        password: document.getElementById("login-password").value,
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
        window.location.href = "index.html";
    } catch (error) {
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

