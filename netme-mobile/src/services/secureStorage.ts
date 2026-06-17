import * as SecureStore from 'expo-secure-store';

const ACCESS_TOKEN_KEY = 'netme_access_token';
const REFRESH_TOKEN_KEY = 'netme_refresh_token';
const USER_KEY = 'netme_user';

class SecureStorage {
  async saveAccessToken(token: string): Promise<void> {
    try {
      await SecureStore.setItemAsync(ACCESS_TOKEN_KEY, token);
    } catch (error) {
      console.error('Failed to save access token:', error);
      throw error;
    }
  }

  async getAccessToken(): Promise<string | null> {
    try {
      return await SecureStore.getItemAsync(ACCESS_TOKEN_KEY);
    } catch (error) {
      console.error('Failed to retrieve access token:', error);
      return null;
    }
  }

  async saveRefreshToken(token: string): Promise<void> {
    try {
      await SecureStore.setItemAsync(REFRESH_TOKEN_KEY, token);
    } catch (error) {
      console.error('Failed to save refresh token:', error);
      throw error;
    }
  }

  async getRefreshToken(): Promise<string | null> {
    try {
      return await SecureStore.getItemAsync(REFRESH_TOKEN_KEY);
    } catch (error) {
      console.error('Failed to retrieve refresh token:', error);
      return null;
    }
  }

  async saveUser(userJson: string): Promise<void> {
    try {
      await SecureStore.setItemAsync(USER_KEY, userJson);
    } catch (error) {
      console.error('Failed to save user:', error);
      throw error;
    }
  }

  async getUser(): Promise<string | null> {
    try {
      return await SecureStore.getItemAsync(USER_KEY);
    } catch (error) {
      console.error('Failed to retrieve user:', error);
      return null;
    }
  }

  async clearAll(): Promise<void> {
    try {
      await SecureStore.deleteItemAsync(ACCESS_TOKEN_KEY);
      await SecureStore.deleteItemAsync(REFRESH_TOKEN_KEY);
      await SecureStore.deleteItemAsync(USER_KEY);
    } catch (error) {
      console.error('Failed to clear secure storage:', error);
      throw error;
    }
  }
}

export const secureStorage = new SecureStorage();
