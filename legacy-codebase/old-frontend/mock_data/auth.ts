import { User, Session, AuthResponse } from '@/lib/types';

// Mock Users
export const mockUsers: User[] = [
  {
    id: '550e8400-e29b-41d4-a716-446655440001',
    email: 'admin@mydreamcampus.edu.tr',
    role: 'admin',
    department: 'Bilgi İşlem',
  },
  {
    id: '550e8400-e29b-41d4-a716-446655440002',
    email: 'ahmet.yilmaz@mydreamcampus.edu.tr',
    role: 'teacher',
    department: 'computer-engineering',
  },
  {
    id: '550e8400-e29b-41d4-a716-446655440003',
    email: 'mehmet.kaya@mydreamcampus.edu.tr',
    role: 'teacher',
    department: 'computer-engineering',
  },
  {
    id: '550e8400-e29b-41d4-a716-446655440004',
    email: 'ayse.demir@mydreamcampus.edu.tr',
    role: 'teacher',
    department: 'mathematics',
  },
  {
    id: '550e8400-e29b-41d4-a716-446655440005',
    email: 'ali.celik@mydreamcampus.edu.tr',
    role: 'student',
    department: 'computer-engineering',
  },
  {
    id: '550e8400-e29b-41d4-a716-446655440006',
    email: 'zeynep.arslan@mydreamcampus.edu.tr',
    role: 'student',
    department: 'computer-engineering',
  },
];

// Mock Sessions
export const mockSessions: Session[] = [
  {
    id: 'session-001',
    device_info: 'Chrome 120 on Windows 11',
    ip_address: '192.168.1.100',
    created_at: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
    last_used_at: new Date(Date.now() - 10 * 60 * 1000).toISOString(),
    is_current: true,
  },
  {
    id: 'session-002',
    device_info: 'Safari on iPhone 15',
    ip_address: '192.168.1.105',
    created_at: new Date(Date.now() - 5 * 60 * 60 * 1000).toISOString(),
    last_used_at: new Date(Date.now() - 30 * 60 * 1000).toISOString(),
    is_current: false,
  },
  {
    id: 'session-003',
    device_info: 'Firefox 121 on macOS Sonoma',
    ip_address: '10.0.0.50',
    created_at: new Date(Date.now() - 10 * 60 * 60 * 1000).toISOString(),
    last_used_at: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
    is_current: false,
  },
];

// Mock Auth Response
export const mockAuthResponse: AuthResponse = {
  access_token: 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiI1NTBlODQwMC1lMjliLTQxZDQtYTcxNi00NDY2NTU0NDAwMDUiLCJlbWFpbCI6ImFsaS5jZWxpa0BteWRyZWFtY2FtcHVzLmVkdS50ciIsInJvbGUiOiJzdHVkZW50In0.abc123',
  expires_in: 86400, // 24 hours in seconds
  user: mockUsers[4], // Ali Çelik (student)
  force_password_change: false,
  message: 'Login successful',
};
