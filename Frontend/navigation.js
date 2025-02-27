document.addEventListener("DOMContentLoaded", () => {
    const forumSection = document.getElementById("forum-section");
    const messagesSection = document.getElementById("messages-section");
    const messagesBtn = document.getElementById("messages-btn");
    const explorePostsBtn = document.getElementById("explore-posts-btn");

    // Check if buttons exist
    if (!messagesBtn) {
        console.error("Messages button not found!");
    } else {
        messagesBtn.addEventListener("click", () => {
            console.log("Switching to messages section");
            forumSection.style.display = "none";
            messagesSection.style.display = "block";
        });
    }

    if (!explorePostsBtn) {
        console.error("Explore posts button not found!");
    } else {
        explorePostsBtn.addEventListener("click", () => {
            console.log("Switching to forum section");
            messagesSection.style.display = "none";
            forumSection.style.display = "block";
        });
    }
});
