const API_BASE_URL = "http://localhost:8088";

document.addEventListener("DOMContentLoaded", () => {
    loadPostDetails();
});

function loadPostDetails() {
    const urlParams = new URLSearchParams(window.location.search);
    const postId = urlParams.get("id");

    if (!postId) {
        console.error("❌ Post ID not found in URL. Check if the URL contains ?id=123");
        return;
    }

    const postUrl = `http://localhost:8088/posts/${postId}`;
    console.log(`🔍 Fetching post from: ${postUrl}`);

    fetch(postUrl)
        .then(response => {
            if (!response.ok) {
                throw new Error(`❌ Post not found (Status: ${response.status})`);
            }
            return response.json();
        })
        .then(post => {
            console.log("✅ Post loaded:", post);
            document.getElementById("post-details").innerHTML = `
                <h2>${post.title}</h2>
                <p>${post.content}</p>
                <p><strong>Posted by:</strong> ${post.nickname}</p>
                ${post.image_path ? `<img src="http://localhost:8088/${post.image_path}" alt="Post Image">` : ""}
                ${post.video_path ? `<video src="http://localhost:8088/${post.video_path}" controls></video>` : ""}
            `;
        })
        .catch(error => console.error("❌ Error loading post:", error));
}


function goBack() {
    window.location.href = "index.html";
}
