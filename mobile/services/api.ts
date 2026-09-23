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

// The app has no cookie jar, so it identifies itself and the backend hands
// the refresh token over in the response body instead of a cookie.
export const CLIENT_HEADERS = {
  'Content-Type': 'application/json',
  'X-Client-Type': 'mobile',
} as const;

export const api = axios.create({
  baseURL: getBaseURL(),
  timeout: 15000,
  headers: { ...CLIENT_HEADERS },
});

const MUTATING_METHODS = new Set(['post', 'put', 'patch', 'delete']);

// Idempotency keys only need to be unique per user (the backend scopes them
// by user id); getRandomValues is used when the runtime has it.
export function newIdempotencyKey(): string {
  const bytes = new Uint8Array(16);
  if (typeof globalThis.crypto?.getRandomValues === 'function') {
    globalThis.crypto.getRandomValues(bytes);
  } else {
    for (let i = 0; i < bytes.length; i++) bytes[i] = Math.floor(Math.random() * 256);
  }
  bytes[6] = (bytes[6] & 0x0f) | 0x40;
  bytes[8] = (bytes[8] & 0x3f) | 0x80;
  const hex = Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('');
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
}

api.interceptors.request.use(
  async (config) => {
    // Set once per logical request: the 401 refresh-and-replay below reuses
    // this config, so the replay carries the same key and cannot, say, book
    // the same meal twice.
    if (
      MUTATING_METHODS.has((config.method ?? '').toLowerCase()) &&
      !config.headers['Idempotency-Key']
    ) {
      config.headers['Idempotency-Key'] = newIdempotencyKey();
    }
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

// Every service answers 403 + FORCE_PASSWORD_CHANGE while the user still
// holds the first-login password; the screen layer decides where to go.
let onForcePasswordChange: (() => void) | null = null;
export const setOnForcePasswordChange = (callback: () => void) => {
  onForcePasswordChange = callback;
};

function isForcedPasswordChange(error: AxiosError): boolean {
  const data = error.response?.data;
  return (
    error.response?.status === 403 &&
    typeof data === 'object' &&
    data !== null &&
    'code' in data &&
    data.code === 'FORCE_PASSWORD_CHANGE'
  );
}

// 'ended' only when the backend refused the refresh token (401). A network
// error, a timeout or a 5xx says nothing about the session, and dropping the
// tokens then would log the user out over a blip.
type RefreshOutcome =
  | { status: 'refreshed'; accessToken: string }
  | { status: 'ended' }
  | { status: 'unavailable' };

function httpStatus(err: unknown): number | undefined {
  if (typeof err !== 'object' || err === null || !('response' in err)) return undefined;
  const { response } = err;
  if (typeof response !== 'object' || response === null || !('status' in response)) return undefined;
  return typeof response.status === 'number' ? response.status : undefined;
}

// Single-flight refresh: avoid stampeding /auth/refresh when many requests
// hit 401 simultaneously. All concurrent 401s wait on the same promise.
let refreshInFlight: Promise<RefreshOutcome> | null = null;

async function refreshOnce(): Promise<RefreshOutcome> {
  if (refreshInFlight) return refreshInFlight;
  refreshInFlight = (async (): Promise<RefreshOutcome> => {
    try {
      const refreshToken = await SecureStore.getItemAsync('refresh_token');
      if (!refreshToken) return { status: 'ended' };
      // Bypass the configured `api` instance to avoid the 401 interceptor
      // running recursively if /auth/refresh itself returns 401.
      const res = await axios.post<{ access_token: string; refresh_token: string }>(
        `${getBaseURL()}/auth/refresh`,
        { refresh_token: refreshToken },
        { timeout: 15000, headers: { ...CLIENT_HEADERS } }
      );
      await SecureStore.setItemAsync('jwt_token', res.data.access_token);
      await SecureStore.setItemAsync('refresh_token', res.data.refresh_token);
      return { status: 'refreshed', accessToken: res.data.access_token };
    } catch (err) {
      return httpStatus(err) === 401 ? { status: 'ended' } : { status: 'unavailable' };
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

    if (isForcedPasswordChange(error)) {
      onForcePasswordChange?.();
      return Promise.reject(error);
    }

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

    const outcome = await refreshOnce();
    if (outcome.status === 'unavailable') {
      return Promise.reject(error);
    }
    if (outcome.status === 'ended') {
      await SecureStore.deleteItemAsync('jwt_token');
      await SecureStore.deleteItemAsync('refresh_token');
      await SecureStore.deleteItemAsync('user_data');
      onUnauthorized?.();
      return Promise.reject(error);
    }

    original._retried = true;
    original.headers = original.headers ?? {};
    original.headers.Authorization = `Bearer ${outcome.accessToken}`;
    return api.request(original);
  }
);

export default api;
