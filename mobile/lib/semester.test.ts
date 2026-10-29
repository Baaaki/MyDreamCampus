import { semesterLabel } from './semester';

describe('semesterLabel', () => {
  it('names the term in Turkish', () => {
    expect(semesterLabel('2026-2027-Fall')).toBe('2026-2027 Güz');
    expect(semesterLabel('2026-2027-Spring')).toBe('2026-2027 Bahar');
    expect(semesterLabel('2026-2027-Summer')).toBe('2026-2027 Yaz');
  });

  it('leaves a name it does not recognise as it is', () => {
    expect(semesterLabel('2024_fall')).toBe('2024_fall');
    expect(semesterLabel('')).toBe('');
  });
});
