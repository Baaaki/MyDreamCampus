// Backend: auth-service/internal/dto/auth_dto.go

export interface User {
  id: string;
  email: string;
  role: 'student' | 'teacher' | 'admin';
  department?: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  user: User;
  force_password_change: boolean;
  message?: string;
}

export interface RefreshResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface ChangePasswordRequest {
  old_password: string;
  new_password: string;
}

export interface ChangePasswordResponse {
  message: string;
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface Session {
  id: string;
  device_info?: string;
  ip_address?: string;
  created_at: string;
  last_used_at: string;
  is_current: boolean;
}

export interface SessionsResponse {
  sessions: Session[];
}

export interface AuthError {
  error: string;
  message?: string;
}
