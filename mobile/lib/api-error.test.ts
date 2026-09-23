import { apiErrorCode, apiErrorMessage } from './api-error';

const axiosError = (data: unknown) => ({ response: { status: 409, data } });

describe('apiErrorMessage', () => {
  it('reads the message of the common {error, code} shape', () => {
    const err = axiosError({ error: 'Bazı gün ve öğünler için zaten rezervasyonunuz var', code: 'RESERVATION_CONFLICTS' });
    expect(apiErrorMessage(err, 'fallback')).toBe('Bazı gün ve öğünler için zaten rezervasyonunuz var');
  });

  it("reads auth's {error: CODE, message} shape", () => {
    const err = axiosError({ error: 'INVALID_OLD_PASSWORD', message: 'Mevcut şifre hatalı' });
    expect(apiErrorMessage(err, 'fallback')).toBe('Mevcut şifre hatalı');
  });

  it('falls back when there is no readable message', () => {
    expect(apiErrorMessage(axiosError({ error: 'INTERNAL_ERROR' }), 'fallback')).toBe('fallback');
    expect(apiErrorMessage(new Error('Network Error'), 'fallback')).toBe('fallback');
    expect(apiErrorMessage(axiosError('<html>'), 'fallback')).toBe('fallback');
  });
});

describe('apiErrorCode', () => {
  it('reads code from either shape', () => {
    expect(apiErrorCode(axiosError({ error: 'metin', code: 'RESERVATION_CONFLICTS' }))).toBe('RESERVATION_CONFLICTS');
    expect(apiErrorCode(axiosError({ error: 'INVALID_OLD_PASSWORD', message: 'metin' }))).toBe('INVALID_OLD_PASSWORD');
    expect(apiErrorCode(new Error('x'))).toBeUndefined();
  });
});
