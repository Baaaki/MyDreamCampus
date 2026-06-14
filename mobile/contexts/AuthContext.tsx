import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { authService } from '@/services/authService';
import { setOnUnauthorized } from '@/services/api';
import type { User } from '@/types/auth.types';

interface AuthContextType {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  setUser: (user: User | null) => void;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const restoreAuth = async () => {
      try {
        const hasToken = await authService.isAuthenticated();
        if (hasToken) {
          const storedUser = await authService.getStoredUser();
          setUser(storedUser);
        }
      } catch (error) {
        console.error('Auth restore error:', error);
        setUser(null);
      } finally {
        setIsLoading(false);
      }
    };

    restoreAuth();
  }, []);

  // Register 401 callback so interceptor can clear auth state
  useEffect(() => {
    setOnUnauthorized(() => setUser(null));
  }, []);

  const logout = useCallback(async () => {
    await authService.logout();
    setUser(null);
  }, []);

  const value = {
    user,
    isLoading,
    isAuthenticated: !!user,
    setUser,
    logout,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

export const useAuthContext = () => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuthContext must be used within an AuthProvider');
  }
  return context;
};
