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
        window.location.href = "auth.html";
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
    if (!titleElement || !contentElement || !categoryElement) {
        console.error("❌ One or more post input fields are missing in the DOM!");
        return;
    }

    const title = titleElement.value.trim();
    const content = contentElement.value.trim();
    const category = categoryElement.value.trim();
    const token = localStorage.getItem("token");

    if (!title || !content) {
        alert("❌ Title and content are required!");
        return;
    }

    const formData = new FormData();
    formData.append("title", title);
    formData.append("content", content);
    formData.append("category", category);
    if (imageInput.files.length > 0) {
        formData.append("image", imageInput.files[0]);
    }
    if (videoInput.files.length > 0) {
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
        const modal = document.getElementById("view-post-modal");
        modal.dataset.postId = post.id;  // Store the post ID in the modal

        const titleElement = document.getElementById("modal-title");
        const authorElement = document.getElementById("modal-author");
        const contentElement = document.getElementById("modal-content");
        const imageElement = document.getElementById("modal-image");
        const videoElement = document.getElementById("modal-video");

        titleElement.textContent = post.title;
        authorElement.textContent = `Posted by: ${post.nickname}`;
        contentElement.textContent = post.content;

        // Display the image if it exists
        if (post.image_path) {
            imageElement.src = `${API_BASE_URL}/${post.image_path}`;
            imageElement.style.display = "block";
        } else {
            imageElement.style.display = "none";
        }

        // Display the video if it exists
        if (post.video_path) {
            videoElement.src = `${API_BASE_URL}/${post.video_path}`;
            videoElement.style.display = "block";
        } else {
            videoElement.style.display = "none";
        }

        modal.style.display = "flex";

        // Load comments for this post
        loadComments(postId);
    })
    .catch(error => console.error("❌ Error loading post:", error));
}


function loadComments(postId) {
    const commentsContainer = document.getElementById("comments-container");
    fetch(`${API_BASE_URL}/comments/all?post_id=${postId}`)
        .then(response => response.json())
        .then(comments => {
            commentsContainer.innerHTML = "";
            comments.forEach(comment => {
                const commentDiv = document.createElement("div");
                commentDiv.classList.add("comment");
                
                // Update to use the nickname from the backend
                commentDiv.innerHTML = `<strong>${comment.nickname}:</strong> ${comment.content}`;
                
                commentsContainer.appendChild(commentDiv);
            });
        })
        .catch(error => console.error("Error loading comments:", error));
}


function createComment() {
    const commentInput = document.getElementById("comment-input");
    const commentText = commentInput.value.trim();
    const postId = document.getElementById("view-post-modal").dataset.postId;  // Get post ID from data attribute
    const token = localStorage.getItem("token");

    console.log("Token:", token); 

    if (!postId) {
        console.error("❌ Post ID is missing!");
        return;
    }
    if (!commentText) {
        alert("❌ Comment content is required!");
        return;
    }

    // Prepare the data to send
    const data = {
        content: commentText,
    };

    // Send the comment to the backend
    fetch(`${API_BASE_URL}/comments?post_id=${postId}`, {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
            "Authorization": `${token}`  // Include token in the Authorization header
        },
        body: JSON.stringify(data)  // Send the comment content in the request body
    })
    .then(response => {
        if (response.status === 401) {
            alert("❌ Unauthorized. Please log in again.");
            window.location.href = "test.html";  // Redirect to login page if unauthorized
            throw new Error("Unauthorized");
        }
        return response.json();  // Parse the response JSON
    })
    .then(() => {
        commentInput.value = "";  // Clear the input field after posting the comment
        loadComments(postId);  // Reload comments after posting
    })
    .catch(error => console.error("❌ Error posting comment:", error));
}


document.addEventListener("DOMContentLoaded", () => {
    checkAuth();
    loadPosts();

    // Comment submission event
    document.getElementById("submit-comment").addEventListener("click", createComment);

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
        window.location.href = "test.html";  // Redirect to login if token is missing
    }
}

// Decode the JWT token to extract user ID
function decodeToken(token) {
    if (!token) return null;

    const payload = token.split('.')[1];
    const decodedPayload = atob(payload);
    const payloadJson = JSON.parse(decodedPayload);

    return payloadJson.user_id;  // Assuming the user ID is stored as 'user_id' in the token
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
    window.location.href = "auth.html";
}
