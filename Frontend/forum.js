const API_BASE_URL = "http://localhost:8088";

document.addEventListener("DOMContentLoaded", () => {
    checkAuth();
    loadPosts();

    document.getElementById("create-post-btn").addEventListener("click", () => {
        document.getElementById("create-post-modal").style.display = "flex";
    });

    document.querySelectorAll(".close").forEach(btn => {
        btn.addEventListener("click", () => {
            closeCreateModal();
            closeViewModal();
        });
    });

    document.getElementById("logout-btn").addEventListener("click", logout);
});

function checkAuth() {
    const token = localStorage.getItem("token");
    if (!token) {
        window.location.href = "test.html";
    }
}

function createPost() {
    // Get input elements
    const titleElement = document.getElementById("post-title");
    const contentElement = document.getElementById("post-content");
    const categoryElement = document.getElementById("post-category");
    const imageInput = document.getElementById("post-image");
    const videoInput = document.getElementById("post-video");

    // Ensure all elements exist before accessing their values
    if (!titleElement || !categoryElement) {
        console.error("❌ One or more required post input fields are missing in the DOM!");
        return;
    }

    const title = titleElement.value.trim();
    const content = contentElement ? contentElement.value.trim() : ""; // Content is optional
    const category = categoryElement.value.trim();
    const token = localStorage.getItem("token");

    // Validate title and category (content can be empty)
    if (!title) {
        alert("❌ Title is required!");
        return;
    }
    if (!category) {
        alert("❌ Category is required!");
        return;
    }

    const formData = new FormData();
    formData.append("title", title);
    formData.append("category", category);
    if (content) {
        formData.append("content", content);
    }
    if (imageInput && imageInput.files.length > 0) {
        formData.append("image", imageInput.files[0]);
    }
    if (videoInput && videoInput.files.length > 0) {
        formData.append("video", videoInput.files[0]);
    }

    fetch(`${API_BASE_URL}/posts`, {
        method: "POST",
        headers: { "Authorization": `Bearer ${token}` },
        body: formData
    })
    .then(response => response.json())
    .then(() => {
        closeCreateModal(); // ✅ Close modal after posting
        loadPosts(); // ✅ Refresh posts
    })
    .catch(error => console.error("❌ Error creating post:", error));
}

// ✅ Close modal function
function closeCreateModal() {
    document.getElementById("create-post-modal").style.display = "none";
}

// ✅ Add event listener for post button after DOM loads
document.addEventListener("DOMContentLoaded", () => {
    const postButton = document.getElementById("post-btn");
    if (postButton) {
        postButton.addEventListener("click", createPost);
    }
});

function loadPosts() {
    const token = localStorage.getItem("token");
    fetch(`${API_BASE_URL}/posts/all`, {
        method: "GET",
        headers: { "Authorization": `Bearer ${token}` }
    })
    .then(response => response.json())
    .then(posts => {
        const container = document.getElementById("posts-container");
        container.innerHTML = "";
        posts.forEach(post => {
            const postElement = document.createElement("div");
            postElement.classList.add("post");

            postElement.innerHTML = `
                <h3>${post.title}</h3>
                <p class="post-author">Posted by: <strong>${post.nickname}</strong></p>
                <p>${post.content.substring(0, 100)}...</p>
                ${post.image_path ? `<img src="${API_BASE_URL}/${post.image_path}" alt="Post Image">` : ""}
                ${post.video_path ? `<video src="${API_BASE_URL}/${post.video_path}" controls></video>` : ""}
                <button onclick="viewPost(${post.id})">View</button>
            `;

            container.appendChild(postElement);
        });
    })
    .catch(error => console.error("Error loading posts:", error));
}

function viewPost(postId) {
    const token = localStorage.getItem("token");
    fetch(`${API_BASE_URL}/post/${postId}`, {
        method: "GET",
        headers: { "Authorization": `Bearer ${token}` }
    })
    .then(response => {
        if (!response.ok) throw new Error(`❌ Post not found (Status: ${response.status})`);
        return response.json();
    })
    .then(post => {
        console.log("✅ Post loaded:", post);

        // ✅ Get correct modal elements
        const modal = document.getElementById("view-post-modal");  // ✅ Correct ID
        const titleElement = document.getElementById("modal-title");
        const authorElement = document.getElementById("modal-author");
        const contentElement = document.getElementById("modal-content");
        const imageElement = document.getElementById("modal-image");
        const videoElement = document.getElementById("modal-video");

        // ❗ Ensure modal elements exist
        if (!modal || !titleElement || !authorElement || !contentElement || !imageElement || !videoElement) {
            console.error("❌ Modal elements not found in the DOM! Check IDs.");
            return;
        }

        // ✅ Update modal content
        titleElement.textContent = post.title;
        authorElement.textContent = `Posted by: ${post.nickname}`;
        contentElement.textContent = post.content;

        // ✅ Handle image
        if (post.image_path) {
            imageElement.src = `${API_BASE_URL}/${post.image_path}`;
            imageElement.style.display = "block";
        } else {
            imageElement.style.display = "none";
        }

        // ✅ Handle video
        if (post.video_path) {
            videoElement.src = `${API_BASE_URL}/${post.video_path}`;
            videoElement.style.display = "block";
        } else {
            videoElement.style.display = "none";
        }

        // ✅ Show modal
        modal.style.display = "flex";
    })
    .catch(error => console.error("❌ Error loading post:", error));
}


window.onclick = function(event) {
    const modal_1 = document.getElementById("view-post-modal");
    const modal = document.getElementById("create-post-modal");
    if (event.target === modal || event.target === modal_1) {
        modal.style.display = "none";
        modal_1.style.display = "none";
    }
};

// ✅ Add event listener for the close button
document.addEventListener("DOMContentLoaded", () => {
    const closeButton = document.querySelector("#create-post-modal .close");
    if (closeButton) {
        closeButton.addEventListener("click", closeCreateModal);
    }
});



function logout() {
    localStorage.removeItem("token");
    window.location.href = "test.html";
}
