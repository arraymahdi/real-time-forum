const API_BASE_URL = "http://localhost:8088";

document.addEventListener("DOMContentLoaded", () => {
    checkAuth();
    loadPosts();

    document.getElementById("create-post-btn").addEventListener("click", () => {
        document.getElementById("create-post-modal").style.display = "flex";
    });

    document.getElementById("post-btn").addEventListener("click", createPost);
    document.getElementById("submit-comment").addEventListener("click", createComment);

    document.querySelectorAll(".close").forEach(btn => {
        btn.addEventListener("click", () => {
            closeCreateModal();
            closeViewModal();
        });
    });

    window.addEventListener("click", (event) => {
        const createModal = document.getElementById("create-post-modal");
        const viewModal = document.getElementById("view-post-modal");
        if (event.target === createModal) createModal.style.display = "none";
        if (event.target === viewModal) viewModal.style.display = "none";
    });

    let isCategorized = false;
    let originalPosts = [];
    document.getElementById("categorize-btn").addEventListener("click", () => {
        const postsContainer = document.getElementById("posts-container");
        if (isCategorized) {
            postsContainer.innerHTML = "";
            originalPosts.forEach(post => postsContainer.appendChild(post));
            isCategorized = false;
            document.getElementById("categorize-btn").textContent = "📂 Categorize Posts";
        } else {
            const posts = Array.from(postsContainer.children);
            if (originalPosts.length === 0) originalPosts = posts.map(post => post.cloneNode(true));

            const categories = {};
            posts.forEach(post => {
                const category = post.getAttribute("data-category") || "Uncategorized";
                if (!categories[category]) categories[category] = [];
                categories[category].push(post);
            });

            postsContainer.innerHTML = "";
            Object.entries(categories).forEach(([category, posts]) => {
                const section = document.createElement("section");
                section.classList.add("category-section");
                section.innerHTML = `<h2 class="category-header">${category}</h2>`;
                const postContainer = document.createElement("div");
                postContainer.classList.add("category-posts");
                posts.forEach(post => postContainer.appendChild(post));
                section.appendChild(postContainer);
                postsContainer.appendChild(section);
            });

            isCategorized = true;
            document.getElementById("categorize-btn").textContent = "📂 Uncategorize Posts";
        }
    });
});

function checkAuth() {
    if (!localStorage.getItem("token")) {
        window.location.href = "index.html";
    }
}

function createPost() {
    const title = document.getElementById("post-title").value.trim();
    const content = document.getElementById("post-content").value.trim();
    const category = document.getElementById("post-category").value.trim();
    const imageInput = document.getElementById("post-image");
    const videoInput = document.getElementById("post-video");
    const token = localStorage.getItem("token");

    if (!title || !content) {
        alert("Title and content are required!");
        return;
    }

    const formData = new FormData();
    formData.append("title", title);
    formData.append("content", content);
    formData.append("category", category);
    if (imageInput.files.length > 0) formData.append("image", imageInput.files[0]);
    if (videoInput.files.length > 0) formData.append("video", videoInput.files[0]);

    fetch(`${API_BASE_URL}/posts`, {
        method: "POST",
        headers: { "Authorization": `Bearer ${token}` },
        body: formData
    })
    .then(response => response.json())
    .then(() => {
        closeCreateModal();
        loadPosts();
    })
    .catch(error => console.error("Error creating post:", error));
}

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
            postElement.setAttribute("data-category", post.category || "Uncategorized");
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
        if (!response.ok) throw new Error("Post not found");
        return response.json();
    })
    .then(post => {
        const modal = document.getElementById("view-post-modal");
        modal.dataset.postId = post.id;
        document.getElementById("modal-title").textContent = post.title;
        document.getElementById("modal-author").textContent = `Posted by: ${post.nickname}`;
        document.getElementById("modal-content").textContent = post.content;
        const image = document.getElementById("modal-image");
        const video = document.getElementById("modal-video");
        if (post.image_path) {
            image.src = `${API_BASE_URL}/${post.image_path}`;
            image.style.display = "block";
        } else {
            image.style.display = "none";
        }
        if (post.video_path) {
            video.src = `${API_BASE_URL}/${post.video_path}`;
            video.style.display = "block";
        } else {
            video.style.display = "none";
        }
        modal.style.display = "flex";
        loadComments(postId);
    })
    .catch(error => console.error("Error loading post:", error));
}

function loadComments(postId) {
    fetch(`${API_BASE_URL}/comments/all?post_id=${postId}`)
    .then(response => response.json())
    .then(comments => {
        const container = document.getElementById("comments-container");
        container.innerHTML = "";
        comments.forEach(comment => {
            const div = document.createElement("div");
            div.classList.add("comment");
            div.innerHTML = `<strong>${comment.nickname}:</strong> ${comment.content}`;
            container.appendChild(div);
        });
    })
    .catch(error => console.error("Error loading comments:", error));
}

function createComment() {
    const content = document.getElementById("comment-input").value.trim();
    const postId = document.getElementById("view-post-modal").dataset.postId;
    const token = localStorage.getItem("token");

    if (!content) {
        alert("Comment content is required!");
        return;
    }

    fetch(`${API_BASE_URL}/comments?post_id=${postId}`, {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
            "Authorization": `${token}`
        },
        body: JSON.stringify({ content })
    })
    .then(response => {
        if (!response.ok) throw new Error("Unauthorized");
        return response.json();
    })
    .then(() => {
        document.getElementById("comment-input").value = "";
        loadComments(postId);
    })
    .catch(error => console.error("Error posting comment:", error));
}

function closeCreateModal() {
    document.getElementById("create-post-modal").style.display = "none";
}

function closeViewModal() {
    document.getElementById("view-post-modal").style.display = "none";
}