import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { mealService } from '@/services/mealService';
import type { CreateReservationRequest } from '@/types/meal.types';

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

export const useCancelReservation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (reservationId: string) => mealService.cancelReservation(reservationId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['my-reservations'] });
    },
  });
};
