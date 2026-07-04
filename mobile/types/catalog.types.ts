// Backend: monolith catalog module DTO'lari (ogrenci kapsami).

export interface SemesterResponse {
  id: string;
  name: string;
  status: string;
  hard_deadline: string;
  created_at: string;
}
