import axios, { AxiosInstance } from 'axios';
import { secureStorage } from './secureStorage';

export interface AuthResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  user: {
    id: string;
    email: string;
    display_name?: string;
    picture_url?: string;
    created_at: string;
    updated_at: string;
  };
}

class AuthService {
  private api: AxiosInstance;
  private isRefreshing = false;
  private refreshSubscribers: ((token: string) => void)[] = [];

  constructor() {
    const apiUrl = process.env.EXPO_PUBLIC_API_URL || 'http://192.168.1.158:8080/v1';

    this.api = axios.create({
      baseURL: apiUrl,
      timeout: 30000,
    });

    this.setupInterceptors();
  }

  private setupInterceptors() {
    this.api.interceptors.request.use(
      async (config) => {
        const token = await secureStorage.getAccessToken();
        if (token) {
          config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
      },
      (error) => Promise.reject(error)
    );

    this.api.interceptors.response.use(
      (response) => response,
      async (error) => {
        const originalRequest = error.config;

        if (
          error.response?.status === 401 &&
          !originalRequest._retry &&
          originalRequest.url !== '/auth/refresh'
        ) {
          originalRequest._retry = true;

          if (!this.isRefreshing) {
            this.isRefreshing = true;
            try {
              const refreshToken = await secureStorage.getRefreshToken();
              if (refreshToken) {
                const response = await this.refresh(refreshToken);
                const { access_token } = response;

                originalRequest.headers.Authorization = `Bearer ${access_token}`;

                this.isRefreshing = false;
                this.onRefreshed(access_token);

                return this.api(originalRequest);
              } else {
                this.isRefreshing = false;
                throw new Error('No refresh token available');
              }
            } catch (refreshError) {
              this.isRefreshing = false;
              await secureStorage.clearAll();
              throw refreshError;
            }
          } else {
            return new Promise((resolve) => {
              this.refreshSubscribers.push((token) => {
                originalRequest.headers.Authorization = `Bearer ${token}`;
                resolve(this.api(originalRequest));
              });
            });
          }
        }

        return Promise.reject(error);
      }
    );
  }

  private onRefreshed(token: string) {
    this.refreshSubscribers.forEach((callback) => callback(token));
    this.refreshSubscribers = [];
  }

  async register(email: string, password: string): Promise<AuthResponse> {
    const response = await this.api.post<AuthResponse>('/auth/register', {
      email,
      password,
    });
    return response.data;
  }

  async login(email: string, password: string): Promise<AuthResponse> {
    const response = await this.api.post<AuthResponse>('/auth/login', {
      email,
      password,
    });
    return response.data;
  }

  async loginWithGoogle(googleIDToken: string): Promise<AuthResponse> {
    const response = await this.api.post<AuthResponse>('/auth/google', {
      id_token: googleIDToken,
    });
    return response.data;
  }

  async refresh(refreshToken: string): Promise<AuthResponse> {
    const response = await this.api.post<AuthResponse>('/auth/refresh', {
      refresh_token: refreshToken,
    });
    return response.data;
  }

  async logout(refreshToken: string, accessToken: string): Promise<void> {
    try {
      await this.api.post(
        '/auth/logout',
        { refresh_token: refreshToken },
        {
          headers: {
            Authorization: `Bearer ${accessToken}`,
          },
        }
      );
    } catch (error) {
      console.error('Logout API call failed:', error);
    }
  }

  async logoutAllDevices(accessToken: string): Promise<void> {
    try {
      await this.api.post(
        '/auth/logout-all-devices',
        {},
        {
          headers: {
            Authorization: `Bearer ${accessToken}`,
          },
        }
      );
    } catch (error) {
      console.error('Logout all devices API call failed:', error);
    }
  }
}

export const authService = new AuthService();
