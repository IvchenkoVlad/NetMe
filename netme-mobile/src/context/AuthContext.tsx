import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { secureStorage } from '../services/secureStorage';
import { authService } from '../services/authService';

export interface User {
  id: string;
  email: string;
  auth_provider: string;
  auth_provider_user_id?: string;
  created_at: string;
  updated_at: string;
}

export interface AuthContextType {
  user: User | null;
  accessToken: string | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  loginWithGoogle: (accessToken: string) => Promise<void>;
  logout: () => Promise<void>;
  clearAuth: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

function getJWTExpiry(token: string): number | null {
  try {
    const base64Url = token.split('.')[1];
    if (!base64Url) return null;
    const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
    const padded = base64 + '==='.slice((base64.length + 3) % 4);
    const payload = JSON.parse(atob(padded));
    return typeof payload.exp === 'number' ? payload.exp : null;
  } catch {
    return null;
  }
}

export const AuthProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [accessToken, setAccessToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    bootstrapAsync();
  }, []);

  const bootstrapAsync = async () => {
    try {
      const savedAccessToken = await secureStorage.getAccessToken();
      const savedRefreshToken = await secureStorage.getRefreshToken();
      const savedUser = await secureStorage.getUser();

      if (!savedAccessToken || !savedRefreshToken || !savedUser) {
        return;
      }

      const expiry = getJWTExpiry(savedAccessToken);
      const nowSeconds = Math.floor(Date.now() / 1000);
      const isExpiredOrExpiringSoon = expiry === null || expiry - nowSeconds < 60;

      if (isExpiredOrExpiringSoon) {
        try {
          const response = await authService.refresh(savedRefreshToken);
          setAccessToken(response.access_token);
          setUser(response.user);
          await secureStorage.saveAccessToken(response.access_token);
          await secureStorage.saveRefreshToken(response.refresh_token);
          await secureStorage.saveUser(JSON.stringify(response.user));
        } catch {
          await secureStorage.clearAll();
        }
      } else {
        setAccessToken(savedAccessToken);
        setUser(JSON.parse(savedUser));
      }
    } catch (error) {
      console.error('Failed to restore session:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const login = async (email: string, password: string) => {
    try {
      setIsLoading(true);
      const response = await authService.login(email, password);

      setUser(response.user);
      setAccessToken(response.access_token);

      await secureStorage.saveAccessToken(response.access_token);
      await secureStorage.saveRefreshToken(response.refresh_token);
      await secureStorage.saveUser(JSON.stringify(response.user));
    } finally {
      setIsLoading(false);
    }
  };

  const register = async (email: string, password: string) => {
    try {
      setIsLoading(true);
      const response = await authService.register(email, password);

      setUser(response.user);
      setAccessToken(response.access_token);

      await secureStorage.saveAccessToken(response.access_token);
      await secureStorage.saveRefreshToken(response.refresh_token);
      await secureStorage.saveUser(JSON.stringify(response.user));
    } finally {
      setIsLoading(false);
    }
  };

  const loginWithGoogle = async (googleAccessToken: string) => {
    try {
      setIsLoading(true);
      const response = await authService.loginWithGoogle(googleAccessToken);
      setUser(response.user);
      setAccessToken(response.access_token);
      await secureStorage.saveAccessToken(response.access_token);
      await secureStorage.saveRefreshToken(response.refresh_token);
      await secureStorage.saveUser(JSON.stringify(response.user));
    } finally {
      setIsLoading(false);
    }
  };

  const logout = async () => {
    try {
      const refreshToken = await secureStorage.getRefreshToken();
      if (refreshToken && accessToken) {
        await authService.logout(refreshToken, accessToken);
      }
    } catch (error) {
      console.error('Logout error:', error);
    } finally {
      await clearAuth();
    }
  };

  const clearAuth = async () => {
    setUser(null);
    setAccessToken(null);
    await secureStorage.clearAll();
  };

  const value: AuthContextType = {
    user,
    accessToken,
    isLoading,
    isAuthenticated: !!accessToken,
    login,
    register,
    loginWithGoogle,
    logout,
    clearAuth,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
