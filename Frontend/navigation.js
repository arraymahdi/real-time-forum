document.addEventListener("DOMContentLoaded", () => {
    const pages = {
        home: document.getElementById("forum-section"),
        messages: document.getElementById("messages-section"),
        auth: document.getElementById("auth-section")
    };

    function navigateTo(page) {
        Object.values(pages).forEach(p => p.style.display = "none");
        if (pages[page]) pages[page].style.display = "block";
    }

    checkAuth();
});

function checkAuth() {
    if (!localStorage.getItem("token")) {
        navigateTo("auth");
    }
}