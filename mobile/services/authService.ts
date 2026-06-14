import api from './api';
import * as SecureStore from 'expo-secure-store';
import type {
  LoginRequest,
  LoginResponse,
  RefreshResponse,
  ChangePasswordRequest,
  ChangePasswordResponse,
  User,
} from '@/types/auth.types';

const TOKEN_KEY = 'jwt_token';
const REFRESH_TOKEN_KEY = 'refresh_token';
const USER_KEY = 'user_data';

async function persistTokens(access?: string, refresh?: string) {
  if (access) await SecureStore.setItemAsync(TOKEN_KEY, access);
  if (refresh) await SecureStore.setItemAsync(REFRESH_TOKEN_KEY, refresh);
}

export const authService = {
  async login(data: LoginRequest): Promise<LoginResponse> {
    const response = await api.post<LoginResponse>('/auth/login', data);
    await persistTokens(response.data.access_token, response.data.refresh_token);
    if (response.data.user) {
      await SecureStore.setItemAsync(USER_KEY, JSON.stringify(response.data.user));
    }
    return response.data;
  },

  async logout(): Promise<void> {
    try {
      await api.post('/auth/logout');
    } catch (error) {
      console.error('Logout API error:', error);
    } finally {
      await SecureStore.deleteItemAsync(TOKEN_KEY);
      await SecureStore.deleteItemAsync(REFRESH_TOKEN_KEY);
      await SecureStore.deleteItemAsync(USER_KEY);
    }
  },

  async changePassword(data: ChangePasswordRequest): Promise<ChangePasswordResponse> {
    const response = await api.post<ChangePasswordResponse>('/auth/password/change', data);
    await persistTokens(response.data.access_token, response.data.refresh_token);
    return response.data;
  },

  // Calls /auth/refresh with the stored refresh token (in body, since
  // mobile axios has no cookie jar). Returns the new access token, or
  // null if the refresh fails.
  async refresh(): Promise<string | null> {
    const refreshToken = await SecureStore.getItemAsync(REFRESH_TOKEN_KEY);
    if (!refreshToken) return null;
    try {
      const response = await api.post<RefreshResponse>('/auth/refresh', {
        refresh_token: refreshToken,
      });
      await persistTokens(response.data.access_token, response.data.refresh_token);
      return response.data.access_token;
    } catch {
      return null;
    }
  },

  async getStoredUser(): Promise<User | null> {
    try {
      const userData = await SecureStore.getItemAsync(USER_KEY);
      if (userData) {
        return JSON.parse(userData) as User;
      }
      return null;
    } catch {
      return null;
    }
  },

  async isAuthenticated(): Promise<boolean> {
    try {
      const token = await SecureStore.getItemAsync(TOKEN_KEY);
      return !!token;
    } catch {
      return false;
    }
  },

  async getToken(): Promise<string | null> {
    try {
      return await SecureStore.getItemAsync(TOKEN_KEY);
    } catch {
      return null;
    }
  },

  async clearAuth(): Promise<void> {
    await SecureStore.deleteItemAsync(TOKEN_KEY);
    await SecureStore.deleteItemAsync(REFRESH_TOKEN_KEY);
    await SecureStore.deleteItemAsync(USER_KEY);
  },
};

export default authService;
