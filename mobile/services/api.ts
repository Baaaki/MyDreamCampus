import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios';
import * as SecureStore from 'expo-secure-store';
import { Platform } from 'react-native';

// EXPO_PUBLIC_API_URL from .env wins; otherwise fall back to simulator defaults.
// Android emulator: 10.0.2.2 aliases host localhost. iOS simulator hits localhost directly.
const getBaseURL = () => {
  const envUrl = process.env.EXPO_PUBLIC_API_URL;
  if (envUrl) return envUrl;
  if (Platform.OS === 'android') return 'http://10.0.2.2/api';
  return 'http://localhost/api';
};

export const api = axios.create({
  baseURL: getBaseURL(),
  timeout: 15000,
  headers: {
    'Content-Type': 'application/json',
  },
});

api.interceptors.request.use(
  async (config) => {
    try {
      const token = await SecureStore.getItemAsync('jwt_token');
      if (token) {
        config.headers.Authorization = `Bearer ${token}`;
      }
    } catch (error) {
      console.error('Error getting token from SecureStore:', error);
    }
    return config;
  },
  (error) => Promise.reject(error)
);

let onUnauthorized: (() => void) | null = null;
export const setOnUnauthorized = (callback: () => void) => {
  onUnauthorized = callback;
};

// Single-flight refresh: avoid stampeding /auth/refresh when many requests
// hit 401 simultaneously. All concurrent 401s wait on the same promise.
let refreshInFlight: Promise<string | null> | null = null;

async function refreshOnce(): Promise<string | null> {
  if (refreshInFlight) return refreshInFlight;
  refreshInFlight = (async () => {
    try {
      const refreshToken = await SecureStore.getItemAsync('refresh_token');
      if (!refreshToken) return null;
      // Bypass the configured `api` instance to avoid the 401 interceptor
      // running recursively if /auth/refresh itself returns 401.
      const res = await axios.post<{ access_token: string; refresh_token: string }>(
        `${getBaseURL()}/auth/refresh`,
        { refresh_token: refreshToken },
        { timeout: 15000, headers: { 'Content-Type': 'application/json' } }
      );
      await SecureStore.setItemAsync('jwt_token', res.data.access_token);
      await SecureStore.setItemAsync('refresh_token', res.data.refresh_token);
      return res.data.access_token;
    } catch {
      return null;
    } finally {
      // Reset on next tick so callers in the same batch share this result.
      setTimeout(() => {
        refreshInFlight = null;
      }, 0);
    }
  })();
  return refreshInFlight;
}

api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const original = error.config as (InternalAxiosRequestConfig & { _retried?: boolean }) | undefined;
    const status = error.response?.status;

    if (status !== 401 || !original) {
      return Promise.reject(error);
    }

    // Don't try to refresh on auth endpoints — 401 there means creds/refresh are invalid.
    const url = original.url ?? '';
    if (url.includes('/auth/login') || url.includes('/auth/refresh')) {
      return Promise.reject(error);
    }

    // Avoid retry loops.
    if (original._retried) {
      await SecureStore.deleteItemAsync('jwt_token');
      await SecureStore.deleteItemAsync('refresh_token');
      await SecureStore.deleteItemAsync('user_data');
      onUnauthorized?.();
      return Promise.reject(error);
    }

    const newAccess = await refreshOnce();
    if (!newAccess) {
      await SecureStore.deleteItemAsync('jwt_token');
      await SecureStore.deleteItemAsync('refresh_token');
      await SecureStore.deleteItemAsync('user_data');
      onUnauthorized?.();
      return Promise.reject(error);
    }

    original._retried = true;
    original.headers = original.headers ?? {};
    original.headers.Authorization = `Bearer ${newAccess}`;
    return api.request(original);
  }
);

export default api;
