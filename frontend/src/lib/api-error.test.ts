import { describe, expect, it } from 'vitest';
import { HTTPError } from 'ky';
import { apiErrorMessage } from './api-error';

function httpError(body: string, status = 400): HTTPError {
  const request = new Request('http://localhost/api/auth/login', { method: 'POST' });
  const response = new Response(body, { status, headers: { 'Content-Type': 'application/json' } });
  return new HTTPError(response, request, {} as never);
}

describe('apiErrorMessage', () => {
  it('prefers the message field (auth shape)', async () => {
    const err = httpError(JSON.stringify({ error: 'INVALID_CREDENTIALS', message: 'Geçersiz e-posta veya şifre' }));
    expect(await apiErrorMessage(err, 'x')).toBe('Geçersiz e-posta veya şifre');
  });

  it('uses the error field when it is prose (service shape)', async () => {
    const err = httpError(JSON.stringify({ error: 'Ders bulunamadı', code: 'COURSE_NOT_FOUND' }));
    expect(await apiErrorMessage(err, 'x')).toBe('Ders bulunamadı');
  });

  it('never shows a bare machine code', async () => {
    const err = httpError(JSON.stringify({ error: 'CSRF_ERROR' }), 403);
    expect(await apiErrorMessage(err, 'İşlem başarısız')).toBe('İşlem başarısız');
  });

  it('falls back on a non-JSON body and on non-HTTP errors', async () => {
    expect(await apiErrorMessage(httpError('<html>'), 'Hata')).toBe('Hata');
    expect(await apiErrorMessage(new TypeError('offline'), 'Hata')).toBe('Hata');
  });
});
