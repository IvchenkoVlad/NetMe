import axios from 'axios';

// Get API URL from environment or default to localhost
const API_URL = process.env.MOBILE_API_URL || 'http://localhost:8080/api/v1';

// Create axios instance
export const apiClient = axios.create({
  baseURL: API_URL,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Add request interceptor (for auth token later)
apiClient.interceptors.request.use(
  (config) => {
    // TODO: Add auth token here when implemented
    // const token = useAuthStore.getState().token;
    // if (token) {
    //   config.headers.Authorization = `Bearer ${token}`;
    // }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Add response interceptor (for error handling)
apiClient.interceptors.response.use(
  (response) => {
    return response.data;
  },
  (error) => {
    console.error('API Error:', error.message);
    return Promise.reject(error);
  }
);

// Simple hello endpoint for testing
export const helloAPI = {
  getHello: async (name: string = 'World') => {
    try {
      const response = await apiClient.get('/hello', {
        params: { name },
      });
      return response;
    } catch (error) {
      console.error('Error calling hello endpoint:', error);
      throw error;
    }
  },
};
