import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { secureStorage } from '../services/secureStorage';
import { authService } from '../services/authService';

export interface User {
  id: string;
  email: string;
  display_name?: string;
  picture_url?: string;
  last_login_at?: string;
  created_at: string;
  updated_at: string;
}

export interface AuthContextType {
  user: User | null;
  accessToken: string | null;
  refreshToken: string | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshAccessToken: () => Promise<boolean>;
  clearAuth: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [accessToken, setAccessToken] = useState<string | null>(null);
  const [refreshToken, setRefreshToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    bootstrapAsync();
  }, []);

  const bootstrapAsync = async () => {
    try {
      const savedAccessToken = await secureStorage.getAccessToken();
      const savedRefreshToken = await secureStorage.getRefreshToken();
      const savedUser = await secureStorage.getUser();

      if (savedAccessToken && savedRefreshToken && savedUser) {
        setAccessToken(savedAccessToken);
        setRefreshToken(savedRefreshToken);
        setUser(JSON.parse(savedUser));
      }
    } catch (error) {
      console.error('Failed to restore token:', error);
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
      setRefreshToken(response.refresh_token);

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
      setRefreshToken(response.refresh_token);

      await secureStorage.saveAccessToken(response.access_token);
      await secureStorage.saveRefreshToken(response.refresh_token);
      await secureStorage.saveUser(JSON.stringify(response.user));
    } finally {
      setIsLoading(false);
    }
  };

  const refreshAccessToken = async (): Promise<boolean> => {
    try {
      if (!refreshToken) {
        return false;
      }

      const response = await authService.refresh(refreshToken);

      setAccessToken(response.access_token);
      setUser(response.user);

      await secureStorage.saveAccessToken(response.access_token);
      await secureStorage.saveUser(JSON.stringify(response.user));

      return true;
    } catch (error) {
      console.error('Token refresh failed:', error);
      await clearAuth();
      return false;
    }
  };

  const logout = async () => {
    try {
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
    setRefreshToken(null);
    await secureStorage.clearAll();
  };

  const value: AuthContextType = {
    user,
    accessToken,
    refreshToken,
    isLoading,
    isAuthenticated: !!accessToken,
    login,
    register,
    logout,
    refreshAccessToken,
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
