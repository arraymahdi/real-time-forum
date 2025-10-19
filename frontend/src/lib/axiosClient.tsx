// lib/axiosClient.ts
import axios from "axios";

const api = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_BASE_URL, // set this in your .env.local
});

// Automatically attach token + user email before each request
api.interceptors.request.use((config) => {
  const token = localStorage.getItem("token");
  const userEmail = localStorage.getItem("user_email"); // make sure you save this on login

  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  if (userEmail) {
    config.headers["User-Email"] = userEmail;
  }

  return config;
});

export default api;
