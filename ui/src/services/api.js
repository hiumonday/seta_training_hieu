import axios from "axios";
import toast from "react-hot-toast";

// REST API client for Go service
const api = axios.create({
  baseURL: "http://localhost:8080/api",
});

// Add JWT token to requests
api.interceptors.request.use((config) => {
  const token = localStorage.getItem("token");
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Handle errors globally
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem("token");
      localStorage.removeItem("user");
      window.location.href = "/login";
    }

    const message = error.response?.data?.error || "An error occurred";
    toast.error(message);
    return Promise.reject(error);
  }
);

export default api;
