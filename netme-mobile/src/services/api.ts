import axios, { AxiosInstance } from 'axios';
import { secureStorage } from './secureStorage';

const API_URL = process.env.EXPO_PUBLIC_API_URL || 'http://localhost:8080/v1';

export const api: AxiosInstance = axios.create({
  baseURL: API_URL,
  timeout: 30000,
});

let isRefreshing = false;
let refreshSubscribers: ((token: string) => void)[] = [];

function onRefreshed(token: string) {
  refreshSubscribers.forEach(cb => cb(token));
  refreshSubscribers = [];
}

api.interceptors.request.use(
  async config => {
    const token = await secureStorage.getAccessToken();
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  error => Promise.reject(error),
);

api.interceptors.response.use(
  response => response,
  async error => {
    const original = error.config;

    if (
      error.response?.status === 401 &&
      !original._retry &&
      original.url !== '/auth/refresh'
    ) {
      original._retry = true;

      if (!isRefreshing) {
        isRefreshing = true;
        try {
          const refreshToken = await secureStorage.getRefreshToken();
          if (!refreshToken) throw new Error('No refresh token');

          // Use raw axios (not the shared `api` instance) to avoid a circular
          // dependency: api.ts would need authService, which imports api.ts.
          const { data } = await axios.post<{ access_token: string; refresh_token: string }>(
            `${API_URL}/auth/refresh`,
            { refresh_token: refreshToken },
          );

          await secureStorage.saveAccessToken(data.access_token);
          await secureStorage.saveRefreshToken(data.refresh_token);

          original.headers.Authorization = `Bearer ${data.access_token}`;
          isRefreshing = false;
          onRefreshed(data.access_token);

          return api(original);
        } catch (refreshError) {
          isRefreshing = false;
          if (axios.isAxiosError(refreshError) && refreshError.response) {
            await secureStorage.clearAll();
          }
          throw refreshError;
        }
      } else {
        return new Promise(resolve => {
          refreshSubscribers.push(token => {
            original.headers.Authorization = `Bearer ${token}`;
            resolve(api(original));
          });
        });
      }
    }

    return Promise.reject(error);
  },
);
