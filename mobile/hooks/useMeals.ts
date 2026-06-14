import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { mealService } from '@/services/mealService';
import type {
  GetMyReservationsParams,
  BatchReservationRequest,
} from '@/types/meal.types';

export const useCafeterias = () => {
  return useQuery({
    queryKey: ['cafeterias'],
    queryFn: () => mealService.getCafeterias(),
    staleTime: 10 * 60 * 1000,
  });
};

export const useMonthlyMenu = (year: number, month: number) => {
  return useQuery({
    queryKey: ['monthly-menu', year, month],
    queryFn: () => mealService.getMonthlyMenu(year, month),
    staleTime: 5 * 60 * 1000,
  });
};

export const useMyReservations = (params?: GetMyReservationsParams) => {
  return useQuery({
    queryKey: ['my-reservations', params],
    queryFn: () => mealService.getMyReservations(params),
    staleTime: 2 * 60 * 1000,
  });
};

export const useCreateBatchReservation = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: BatchReservationRequest) => mealService.createBatchReservation(data),
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
