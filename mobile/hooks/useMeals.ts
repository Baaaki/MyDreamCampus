import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { mealService } from '@/services/mealService';
import { paymentService } from '@/services/paymentService';
import type {
  BatchReservationRequest,
  CreateReservationRequest,
  Reservation,
} from '@/types/meal.types';
import type { ConfirmPaymentRequest } from '@/types/payment.types';

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
      // Rows land as `pending` and stay so until the card is confirmed.
      queryClient.invalidateQueries({ queryKey: ['my-reservations'] });
    },
  });
};

export const useConfirmPayment = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ paymentId, card }: { paymentId: string; card: ConfirmPaymentRequest }) =>
      paymentService.confirm(paymentId, card),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['my-reservations'] });
    },
  });
};

/**
 * Where a checkout's reservations stand. Payment settles the card at once,
 * but meal learns the outcome from an event, so the reservations stay
 * pending for a moment after the card is accepted or declined.
 */
export function checkoutOutcome(
  reservationIds: string[],
  reservations: Reservation[]
): 'confirmed' | 'cancelled' | 'pending' {
  const mine = reservations.filter((r) => reservationIds.includes(r.id));
  if (mine.length < reservationIds.length) return 'pending';
  if (mine.some((r) => r.status === 'cancelled' || r.status === 'expired')) return 'cancelled';
  return mine.every((r) => r.status === 'confirmed') ? 'confirmed' : 'pending';
}

const CHECKOUT_POLL_MS = 1500;
// Meal usually confirms within seconds; past this the student is sent to
// the reservation list instead of watching a spinner.
const CHECKOUT_TIMEOUT_MS = 30_000;

/**
 * Polls the reservations of a paid checkout until meal confirms or cancels
 * them, or until the timeout. startedAt is when the card was confirmed.
 */
export const useCheckoutOutcome = (reservationIds: string[], startedAt: number | null) => {
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: ['checkout-reservations', ...reservationIds],
    queryFn: async () => {
      const data = await mealService.getMyReservations();
      // The same list the screen shows; keep it current while polling.
      queryClient.setQueryData(['my-reservations'], data);
      return data;
    },
    enabled: startedAt !== null && reservationIds.length > 0,
    retry: false,
    refetchInterval: (q) => {
      const { data, dataUpdatedAt, errorUpdatedAt } = q.state;
      if (Math.max(dataUpdatedAt, errorUpdatedAt) - (startedAt ?? 0) > CHECKOUT_TIMEOUT_MS) {
        return false;
      }
      if (data && checkoutOutcome(reservationIds, data.reservations ?? []) !== 'pending') {
        return false;
      }
      return CHECKOUT_POLL_MS;
    },
  });
  const outcome = query.data
    ? checkoutOutcome(reservationIds, query.data.reservations ?? [])
    : 'pending';
  // Failed polls count too, so an unreachable meal service ends the wait.
  const timedOut =
    startedAt !== null &&
    Math.max(query.dataUpdatedAt, query.errorUpdatedAt) - startedAt > CHECKOUT_TIMEOUT_MS;
  return { outcome, timedOut };
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
