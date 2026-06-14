import { Cafeteria, MyReservationsResponse, QRResponse } from '@/lib/types';

// Mock Cafeterias (from meal-service)
export const mockCafeterias: Cafeteria[] = [
  {
    id: '550e8400-e29b-41d4-a716-446655440501',
    name: 'Merkez Kafeterya',
    location: 'Ana Kampüs - A Blok Zemin Kat',
    has_vegan_menu: true,
    serves_dinner: true,
    is_active: true,
    created_at: '2020-01-15T10:00:00Z',
    updated_at: '2025-01-08T10:00:00Z',
  },
  {
    id: '550e8400-e29b-41d4-a716-446655440502',
    name: 'Mühendislik Kafeterya',
    location: 'Mühendislik Fakültesi - C Blok 1. Kat',
    has_vegan_menu: true,
    serves_dinner: false,
    is_active: true,
    created_at: '2020-01-15T10:00:00Z',
    updated_at: '2025-01-08T10:00:00Z',
  },
  {
    id: '550e8400-e29b-41d4-a716-446655440503',
    name: 'Sosyal Tesis Kafeterya',
    location: 'Sosyal Tesisler Binası',
    has_vegan_menu: false,
    serves_dinner: true,
    is_active: true,
    created_at: '2020-01-15T10:00:00Z',
    updated_at: '2025-01-08T10:00:00Z',
  },
  {
    id: '550e8400-e29b-41d4-a716-446655440504',
    name: 'Yemekhane (Eski)',
    location: 'Eski Kampüs',
    has_vegan_menu: false,
    serves_dinner: false,
    is_active: false,
    created_at: '2020-01-15T10:00:00Z',
    updated_at: '2025-01-08T10:00:00Z',
  },
];

// Generate dates for next 7 days
const today = new Date();
const generateFutureDates = (days: number): Date[] => {
  return Array.from({ length: days }, (_, i) => {
    const date = new Date(today);
    date.setDate(today.getDate() + i);
    return date;
  });
};

const futureDates = generateFutureDates(7);

// Mock My Reservations Response (from meal-service)
export const mockMyReservationsResponse: MyReservationsResponse = {
  reservations: [
    // Today's reservations
    {
      id: '550e8400-e29b-41d4-a716-446655440601',
      date: futureDates[0].toISOString().split('T')[0],
      meal_time: 'lunch',
      menu_type: 'normal',
      cafeteria_name: 'Merkez Kafeterya',
      cafeteria: {
        id: '550e8400-e29b-41d4-a716-446655440501',
        name: 'Merkez Kafeterya',
        location: 'Ana Kampüs - A Blok Zemin Kat',
      },
      status: 'confirmed',
      is_used: false,
      created_at: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
    },
    {
      id: '550e8400-e29b-41d4-a716-446655440602',
      date: futureDates[0].toISOString().split('T')[0],
      meal_time: 'dinner',
      menu_type: 'normal',
      cafeteria_name: 'Merkez Kafeterya',
      cafeteria: {
        id: '550e8400-e29b-41d4-a716-446655440501',
        name: 'Merkez Kafeterya',
        location: 'Ana Kampüs - A Blok Zemin Kat',
      },
      status: 'confirmed',
      is_used: false,
      created_at: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
    },
    // Tomorrow's reservations
    {
      id: '550e8400-e29b-41d4-a716-446655440603',
      date: futureDates[1].toISOString().split('T')[0],
      meal_time: 'lunch',
      menu_type: 'vegan',
      cafeteria_name: 'Mühendislik Kafeterya',
      cafeteria: {
        id: '550e8400-e29b-41d4-a716-446655440502',
        name: 'Mühendislik Kafeterya',
        location: 'Mühendislik Fakültesi - C Blok 1. Kat',
      },
      status: 'confirmed',
      is_used: false,
      created_at: new Date(Date.now() - 20 * 60 * 60 * 1000).toISOString(),
    },
    // Past reservations (used)
    {
      id: '550e8400-e29b-41d4-a716-446655440604',
      date: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString().split('T')[0],
      meal_time: 'lunch',
      menu_type: 'normal',
      cafeteria_name: 'Merkez Kafeterya',
      cafeteria: {
        id: '550e8400-e29b-41d4-a716-446655440501',
        name: 'Merkez Kafeterya',
        location: 'Ana Kampüs - A Blok Zemin Kat',
      },
      status: 'confirmed',
      is_used: true,
      created_at: new Date(Date.now() - 48 * 60 * 60 * 1000).toISOString(),
    },
    {
      id: '550e8400-e29b-41d4-a716-446655440605',
      date: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString().split('T')[0],
      meal_time: 'dinner',
      menu_type: 'normal',
      cafeteria_name: 'Merkez Kafeterya',
      cafeteria: {
        id: '550e8400-e29b-41d4-a716-446655440501',
        name: 'Merkez Kafeterya',
        location: 'Ana Kampüs - A Blok Zemin Kat',
      },
      status: 'confirmed',
      is_used: true,
      created_at: new Date(Date.now() - 48 * 60 * 60 * 1000).toISOString(),
    },
    // Pending reservation
    {
      id: '550e8400-e29b-41d4-a716-446655440606',
      date: futureDates[3].toISOString().split('T')[0],
      meal_time: 'lunch',
      menu_type: 'normal',
      cafeteria_name: 'Sosyal Tesis Kafeterya',
      cafeteria: {
        id: '550e8400-e29b-41d4-a716-446655440503',
        name: 'Sosyal Tesis Kafeterya',
        location: 'Sosyal Tesisler Binası',
      },
      status: 'pending',
      is_used: false,
      created_at: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
    },
    // Cancelled reservation
    {
      id: '550e8400-e29b-41d4-a716-446655440607',
      date: futureDates[2].toISOString().split('T')[0],
      meal_time: 'dinner',
      menu_type: 'normal',
      cafeteria_name: 'Merkez Kafeterya',
      status: 'cancelled',
      is_used: false,
      created_at: new Date(Date.now() - 30 * 60 * 60 * 1000).toISOString(),
    },
  ],
  summary: {
    total: 7,
    confirmed: 5,
    pending: 1,
    used: 2,
    cancelled: 1,
  },
};

// Mock QR Codes for today (from meal-service)
export const mockQRResponses: QRResponse[] = [
  {
    cafeteria_id: '550e8400-e29b-41d4-a716-446655440501',
    cafeteria_name: 'Merkez Kafeterya',
    date: futureDates[0].toISOString().split('T')[0],
    meal_time: 'lunch',
    qr_payload: 'QR-LUNCH-001-' + futureDates[0].toISOString().split('T')[0],
    valid_time_window: {
      start: '11:00',
      end: '13:00',
    },
  },
  {
    cafeteria_id: '550e8400-e29b-41d4-a716-446655440501',
    cafeteria_name: 'Merkez Kafeterya',
    date: futureDates[0].toISOString().split('T')[0],
    meal_time: 'dinner',
    qr_payload: 'QR-DINNER-001-' + futureDates[0].toISOString().split('T')[0],
    valid_time_window: {
      start: '17:00',
      end: '19:00',
    },
  },
  {
    cafeteria_id: '550e8400-e29b-41d4-a716-446655440502',
    cafeteria_name: 'Mühendislik Kafeterya',
    date: futureDates[0].toISOString().split('T')[0],
    meal_time: 'lunch',
    qr_payload: 'QR-LUNCH-002-' + futureDates[0].toISOString().split('T')[0],
    valid_time_window: {
      start: '11:00',
      end: '13:00',
    },
  },
];

// Helper: Get available reservation slots
export const getAvailableSlots = (cafeteriaId: string, date: string): {
  lunch: number;
  dinner: number;
} => {
  const cafeteria = mockCafeterias.find(c => c.id === cafeteriaId);
  if (!cafeteria) return { lunch: 0, dinner: 0 };

  // Mock capacity: 500 for Merkez, 300 for Mühendislik, 200 for Sosyal Tesis
  const capacity = cafeteria.name.includes('Merkez') ? 500 :
                   cafeteria.name.includes('Mühendislik') ? 300 : 200;

  const reservations = mockMyReservationsResponse.reservations.filter(
    r => r.cafeteria?.id === cafeteriaId && r.date === date && r.status === 'confirmed'
  );

  const lunchCount = reservations.filter(r => r.meal_time === 'lunch').length;
  const dinnerCount = reservations.filter(r => r.meal_time === 'dinner').length;

  return {
    lunch: capacity - lunchCount,
    dinner: cafeteria.serves_dinner ? capacity - dinnerCount : 0,
  };
};
