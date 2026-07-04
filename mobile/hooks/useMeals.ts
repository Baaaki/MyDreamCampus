import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { mealService } from '@/services/mealService';
import type { BatchReservationRequest, CreateReservationRequest } from '@/types/meal.types';

export const useCafeterias = () =>
  useQuery({
    queryKey: ['cafeterias'],
    queryFn: () => mealService.getCafeterias(),
    staleTime: 30 * 60 * 1000,
  });

export const useMonthlyMenu = (year: number, month: number) =>
  useQuery({
    queryKey: ['monthly-menu', year, month],
    queryFn: () => mealService.getMonthlyMenu(year, month),
    staleTime: 30 * 60 * 1000,
  });

export const useMyReservations = () =>
  useQuery({
    queryKey: ['my-reservations'],
    queryFn: () => mealService.getMyReservations(),
    staleTime: 60 * 1000,
  });

export const useCreateReservation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateReservationRequest) => mealService.createReservation(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['my-reservations'] });
    },
  });
};

export const useCreateBatchReservation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: BatchReservationRequest) => mealService.createBatchReservation(data),
    onSuccess: () => {
      // Rows land as `pending`; the mock payment confirms them ~2s later via an
      // event. Invalidate now for the optimistic list, then again after the
      // webhook window so the status flips to `confirmed` without a manual pull.
      queryClient.invalidateQueries({ queryKey: ['my-reservations'] });
      setTimeout(() => {
        queryClient.invalidateQueries({ queryKey: ['my-reservations'] });
      }, 2500);
    },
  });
};

export const useCancelReservation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (reservationId: string) => mealService.cancelReservation(reservationId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['my-reservations'] });
    },
  });
};

// Toplu iptal: her id icin tekil iptal cagrilir. allSettled ile bir randevunun
// reddi (ornek: kilit suresi gecmis) digerlerini iptal etmeyi engellemez;
// kac tanesinin basarisiz oldugunu geri doneriz ki UI bilgi verebilsin.
export const useCancelReservations = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (ids: string[]) => {
      const results = await Promise.allSettled(
        ids.map((id) => mealService.cancelReservation(id))
      );
      const failed = results.filter((r) => r.status === 'rejected').length;
      return { total: ids.length, failed };
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['my-reservations'] });
    },
  });
};
