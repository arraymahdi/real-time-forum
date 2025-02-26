const API_URL = 'http://localhost:8088';
let token = localStorage.getItem('token');

document.addEventListener('DOMContentLoaded', () => {
    checkAuth();
});

function checkAuth() {
    if (token) {
        document.getElementById('auth-page').classList.add('hidden');
        document.getElementById('forum-page').classList.remove('hidden');
        loadPosts();
    } else {
        document.getElementById('auth-page').classList.remove('hidden');
        document.getElementById('forum-page').classList.add('hidden');
    }
}

async function register() {
    const user = {
        nickname: document.getElementById('register-nickname').value,
        age: document.getElementById('register-age').value,
        gender: document.getElementById('register-gender').value,
        first_name: document.getElementById('register-first-name').value,
        last_name: document.getElementById('register-last-name').value,
        email: document.getElementById('register-email').value,
        password: document.getElementById('register-password').value,
    };
    
    const response = await fetch(`${API_URL}/register`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(user)
    });
    
    if (response.ok) {
        alert('Registration successful!');
    } else {
        alert('Registration failed!');
    }
}

async function login() {
    const credentials = {
        email: document.getElementById('login-username').value,
        password: document.getElementById('login-password').value
    };
    
    const response = await fetch(`${API_URL}/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(credentials)
    });
    
    const data = await response.json();
    if (response.ok) {
        localStorage.setItem('token', data.token);
        token = data.token;
        checkAuth();
    } else {
        alert('Login failed!');
    }
}

async function createPost() {
    const postContent = document.getElementById('post-content').value;
    const postImage = document.getElementById('post-image').files[0];
    const formData = new FormData();
    formData.append('content', postContent);
    if (postImage) {
        formData.append('image', postImage);
    }
    
    const response = await fetch(`${API_URL}/posts`, {
        method: 'POST',
        headers: {
            'Authorization': `Bearer ${token}`
        },
        body: formData
    });
    
    if (response.ok) {
        loadPosts();
    } else {
        alert('Failed to create post!');
    }
}

async function loadPosts() {
    const response = await fetch(`${API_URL}/posts/all`, {
        headers: { 'Authorization': `Bearer ${token}` }
    });
    const posts = await response.json();
    document.getElementById('posts').innerHTML = posts.map(post => `<p>${post.content}<br><img src="${post.image || ''}" width="200"></p>`).join('');
}

function logout() {
    localStorage.removeItem('token');
    token = null;
    checkAuth();
}
