import { api } from './api';

export interface AuthResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  user: {
    id: string;
    email: string;
    auth_provider: string;
    auth_provider_user_id?: string;
    created_at: string;
    updated_at: string;
  };
}

class AuthService {
  async register(email: string, password: string): Promise<AuthResponse> {
    const { data } = await api.post<AuthResponse>('/auth/register', { email, password });
    return data;
  }

  async login(email: string, password: string): Promise<AuthResponse> {
    const { data } = await api.post<AuthResponse>('/auth/login', { email, password });
    return data;
  }

  async loginWithGoogle(googleIDToken: string): Promise<AuthResponse> {
    const { data } = await api.post<AuthResponse>('/auth/google', { id_token: googleIDToken });
    return data;
  }

  async refresh(refreshToken: string): Promise<AuthResponse> {
    const { data } = await api.post<AuthResponse>('/auth/refresh', { refresh_token: refreshToken });
    return data;
  }

  async logout(refreshToken: string, accessToken: string): Promise<void> {
    try {
      await api.post(
        '/auth/logout',
        { refresh_token: refreshToken },
        { headers: { Authorization: `Bearer ${accessToken}` } },
      );
    } catch (error) {
      console.error('Logout API call failed:', error);
    }
  }

  async deleteAccount(): Promise<void> {
    await api.delete('/me');
  }
}

export const authService = new AuthService();
