import api from './api';
import * as SecureStore from 'expo-secure-store';
import type {
  LoginRequest,
  LoginResponse,
  RefreshResponse,
  ChangePasswordRequest,
  ChangePasswordResponse,
  User,
  DemoAccount,
} from '@/types/auth.types';

const TOKEN_KEY = 'jwt_token';
const REFRESH_TOKEN_KEY = 'refresh_token';
const USER_KEY = 'user_data';

export const STUDENT_ONLY_MESSAGE =
  'Mobil uygulama öğrenciler içindir; öğretmen ve yöneticiler web panelini kullanır.';

export class StudentOnlyError extends Error {
  constructor() {
    super(STUDENT_ONLY_MESSAGE);
    this.name = 'StudentOnlyError';
  }
}

async function persistTokens(access?: string, refresh?: string) {
  if (access) await SecureStore.setItemAsync(TOKEN_KEY, access);
  if (refresh) await SecureStore.setItemAsync(REFRESH_TOKEN_KEY, refresh);
}

export const authService = {
  async login(data: LoginRequest): Promise<LoginResponse> {
    const response = await api.post<LoginResponse>('/auth/login', data);
    // The app only calls student endpoints; any other role would get a 403
    // on every screen. Nothing is persisted locally, and the session the
    // server just opened is ended with the tokens still in hand (nothing is
    // in SecureStore yet for the request interceptor to attach).
    if (response.data.user && response.data.user.role !== 'student') {
      try {
        await api.post(
          '/auth/logout',
          { refresh_token: response.data.refresh_token },
          { headers: { Authorization: `Bearer ${response.data.access_token}` } }
        );
      } catch {
        // Best effort: the user is turned away either way.
      }
      await authService.clearAuth();
      throw new StudentOnlyError();
    }
    await persistTokens(response.data.access_token, response.data.refresh_token);
    if (response.data.user) {
      await SecureStore.setItemAsync(USER_KEY, JSON.stringify(response.data.user));
    }
    return response.data;
  },

  async logout(): Promise<void> {
    try {
      // Sent in the body because there is no cookie to carry it; without it
      // the server cannot end the session and the refresh token outlives
      // the logout by a day.
      const refreshToken = await SecureStore.getItemAsync(REFRESH_TOKEN_KEY);
      await api.post('/auth/logout', refreshToken ? { refresh_token: refreshToken } : {});
    } catch (error) {
      console.error('Logout API error:', error);
    } finally {
      await SecureStore.deleteItemAsync(TOKEN_KEY);
      await SecureStore.deleteItemAsync(REFRESH_TOKEN_KEY);
      await SecureStore.deleteItemAsync(USER_KEY);
    }
  },

  async changePassword(data: ChangePasswordRequest): Promise<ChangePasswordResponse> {
    const response = await api.post<ChangePasswordResponse>('/auth/change-password', data);
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

  // A non-student left in storage (older build, tampered store) would see
  // only 403s, so drop the whole session instead of restoring it.
  async restoreStudent(): Promise<User | null> {
    const user = await authService.getStoredUser();
    if (user && user.role !== 'student') {
      await authService.clearAuth();
      return null;
    }
    return user;
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

  async getDemoAccounts(): Promise<DemoAccount[]> {
    try {
      const response = await api.get<DemoAccount[]>('/auth/demo-accounts');
      return Array.isArray(response.data) ? response.data : [];
    } catch {
      return [];
    }
  },
  async clearAuth(): Promise<void> {
    await SecureStore.deleteItemAsync(TOKEN_KEY);
    await SecureStore.deleteItemAsync(REFRESH_TOKEN_KEY);
    await SecureStore.deleteItemAsync(USER_KEY);
  },
};

export default authService;
