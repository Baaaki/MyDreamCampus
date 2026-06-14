import { useMutation, useQueryClient } from '@tanstack/react-query';
import { authService } from '@/services/authService';
import type { LoginRequest, ChangePasswordRequest } from '@/types/auth.types';

/**
 * Login mutation hook
 */
export const useLogin = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: LoginRequest) => authService.login(data),
    onSuccess: (data) => {
      queryClient.setQueryData(['user'], data.user);
    },
  });
};

/**
 * Logout mutation hook
 */
export const useLogout = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => authService.logout(),
    onSuccess: () => {
      queryClient.clear();
    },
  });
};

/**
 * Change password mutation hook
 */
export const useChangePassword = () => {
  return useMutation({
    mutationFn: (data: ChangePasswordRequest) => authService.changePassword(data),
  });
};
