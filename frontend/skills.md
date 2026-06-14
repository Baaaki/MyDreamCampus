# Frontend — React + Vite (AI Talimati)

React 19 + Vite + react-router v7 + TanStack Query + ky + shadcn/ui + Tailwind 4. `frontend/src/**` icinde calisirken bu dosya zorunlu okumadir.

> **Onemli:** Bu Next.js DEGIL. Vite. `'use client'` yok, `next/*` yok.

---

## 1. Sert Kurallar (asla ihlal etme)

- **Paket yoneticisi**: `bun` — `npm`, `npx`, `yarn` YAPMA. Tip kontrolu icin `bun tsc --noEmit`.
- **Routing**: `react-router` v7 — `next/navigation`, `next/link` import etme.
- **HTTP**: `ky` (`src/lib/api-client.ts`) — `fetch` direkt veya `axios` kullanma.
- **Server state**: `@tanstack/react-query` — `useEffect` icinde fetch YAPMA.
- **`'use client'` direktifi YOK** — Vite SSR yapmiyor.
- **Env var prefix**: `VITE_` — `process.env` YAPMA, `import.meta.env.VITE_X` kullan.
- **Alias**: `@/` -> `src/` — relative path (`../../components/...`) YAPMA.
- **CSRF**: `api-client.ts` cookie'den `X-CSRF-Token` ekliyor — manuel ekleme.

---

## 2. Dosya Yapisi (sabit)

```
frontend/src/
├── pages/                     # Sayfa componentleri
│   ├── auth/                  # login, register
│   ├── student/               # ogrenci sayfalari
│   ├── teacher/               # ogretmen sayfalari
│   ├── admin/                 # admin sayfalari
│   └── not-found.tsx
├── components/
│   ├── ui/                    # shadcn (Button, Card, Dialog vs.) — DUZENLEME
│   ├── layout/                # AppLayout, Sidebar, Header (Outlet kullanir)
│   ├── providers/             # ThemeProvider, AuthProvider
│   ├── auth-guard.tsx         # protected route wrapper
│   ├── error-boundary.tsx     # ErrorBoundary
│   └── {feature}/             # feature-spesifik componentler
├── lib/
│   ├── api-client.ts          # ky instance + 401 refresh + CSRF
│   ├── api-types.ts           # OpenAPI'den uretilen — DOKUNMA
│   ├── services/              # API call fonksiyonlari (auth-service.ts vs.)
│   ├── types.ts               # Manuel TypeScript tipleri
│   ├── utils.ts               # cn() helper
│   └── constants.ts           # Sabitler (roller, urunler vs.)
├── routes.tsx                 # Tum route tanimlari
├── main.tsx                   # Entry (provider'lar burada)
├── App.tsx                    # Router root
└── index.css                  # Tailwind + global CSS
```

---

## 3. Yeni Sayfa Workflow

```
1. Type tanimi:    src/lib/types.ts (Request/Response tipleri)
2. Service:        src/lib/services/{feature}-service.ts (Bolum 6 sablonu)
3. Hook (varsa):   src/hooks/use{Feature}.ts (TanStack Query wrapper)
4. Page bilesen:   src/pages/{role}/{feature}/index.tsx (Bolum 7 sablonu)
5. Route ekle:     src/routes.tsx
6. Tip kontrol:    bun tsc --noEmit
7. Browser test:   bun run dev
8. Commit:         feat(frontend): add {feature} page
```

---

## 4. State Karar Matrisi (zorunlu uy)

| Veri turu | Kullan |
|---|---|
| Server'dan gelen data | **TanStack Query** (`useQuery`, `useMutation`) |
| Form input degerleri | `useState` (basit) veya `react-hook-form` (kompleks/zod) |
| URL filtreleri (sayfalama, search) | `useSearchParams` (react-router) |
| Modal/sidebar acik-kapali | `useState` |
| Tema, kullanici bilgisi | Context (`ThemeProvider`, `AuthProvider`) |
| Hesaplanmis deger | `useMemo` |
| Side effect (DOM, listener) | `useEffect` — **fetch icin DEGIL** |

**YAPMA:**
- Server data'yi `useState` + `useEffect` ile cek
- Global state icin Redux/Zustand ekle (zaten yok, ekleme)
- URL state'i React state'inde tut (refresh sonrasi kayboluyor)

---

## 5. Routing Hizli Referans

```tsx
import { Link, useNavigate, useLocation, useParams, useSearchParams } from 'react-router';

// Link
<Link to="/students">Ogrenciler</Link>          // ✅ to (href DEGIL)
<Link to="/students/123" state={{ from: 'list' }}>Detay</Link>

// Programmatic
const navigate = useNavigate();
navigate('/students');
navigate(-1);                                    // geri
navigate('/login', { replace: true });           // history replace

// URL parametre
const { id } = useParams<{ id: string }>();      // /students/:id

// Query parametre
const [searchParams, setSearchParams] = useSearchParams();
const page = Number(searchParams.get('page') ?? '1');
setSearchParams({ page: '2', q: 'ali' });

// Aktif route
const { pathname } = useLocation();
```

**YAPMA:**
- `<a href="/internal-route">` — full page reload
- `window.location.href` — SPA navigasyonu kir
- `useRouter` — Next.js'e ait, burada yok

---

## 6. Service Sablonu (`src/lib/services/`)

```ts
// src/lib/services/xxx-service.ts
import { xxxApi } from '@/lib/api-client';
import type {
  CreateXxxRequest,
  XxxResponse,
  XxxListResponse,
} from '@/lib/types';

export const xxxService = {
  async list(params?: { page?: number; q?: string }): Promise<XxxListResponse> {
    return xxxApi
      .get('', { searchParams: params })
      .json<XxxListResponse>();
  },

  async getById(id: string): Promise<XxxResponse> {
    return xxxApi.get(id).json<XxxResponse>();
  },

  async create(data: CreateXxxRequest): Promise<XxxResponse> {
    return xxxApi.post('', { json: data }).json<XxxResponse>();
  },

  async update(id: string, data: Partial<CreateXxxRequest>): Promise<XxxResponse> {
    return xxxApi.put(id, { json: data }).json<XxxResponse>();
  },

  async remove(id: string): Promise<void> {
    await xxxApi.delete(id);
  },
};
```

**ky kurallari:**
- POST/PUT body: `{ json: data }` (`{ body: JSON.stringify(...) }` DEGIL — ky otomatik halleder)
- Query string: `{ searchParams: { page: 1 } }`
- Response parse: `.json<T>()` (`.then(r => r.json())` DEGIL)
- 401 → `api-client.ts` otomatik refresh + redirect — kendin handle etme
- Hata throw eder — try/catch sadece kullaniciya custom mesaj icin

**YAPMA:**
- `ky.post('https://...')` — yeni instance kurma, `xxxApi` kullan
- Path'a `/` ile basla — `prefixUrl` zaten `/api/xxx`, `xxxApi.get('list')` yeterli
- Trailing slash ekleme — `api-client.ts` kaldiriyor zaten

---

## 7. Sayfa Component Sablonu

```tsx
// src/pages/{role}/{feature}/index.tsx
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router';
import { Loader2, AlertCircle } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { xxxService } from '@/lib/services/xxx-service';

export default function XxxListPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['xxx-list'],
    queryFn: () => xxxService.list(),
    staleTime: 5 * 60 * 1000,  // 5 dk
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => xxxService.remove(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['xxx-list'] });
    },
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-8">
        <Loader2 className="h-6 w-6 animate-spin" />
      </div>
    );
  }

  if (isError) {
    return (
      <Card>
        <CardContent className="flex items-center gap-2 p-6 text-destructive">
          <AlertCircle className="h-5 w-5" />
          <span>{error instanceof Error ? error.message : 'Bir hata olustu'}</span>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Liste</CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        {data?.items.map((item) => (
          <div key={item.id} className="flex items-center justify-between">
            <span>{item.name}</span>
            <Button
              variant="destructive"
              size="sm"
              onClick={() => deleteMutation.mutate(item.id)}
              disabled={deleteMutation.isPending}
            >
              Sil
            </Button>
          </div>
        ))}
      </CardContent>
    </Card>
  );
}
```

**Sayfa kurallari:**
- Default export → component adi PascalCase
- Loading state: `Loader2` + `animate-spin`
- Error state: `AlertCircle` + Turkce mesaj
- Empty state: kullaniciya ne yapacagini soyle ("Henuz X yok, eklemek icin...")
- Mutation sonrasi `queryClient.invalidateQueries` (re-fetch)

---

## 8. Form Sablonu (react-hook-form + zod)

```tsx
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useMutation } from '@tanstack/react-query';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { xxxService } from '@/lib/services/xxx-service';

const schema = z.object({
  name: z.string().min(2, 'En az 2 karakter').max(100, 'En fazla 100 karakter'),
  email: z.string().email('Gecerli bir e-posta giriniz'),
});

type FormValues = z.infer<typeof schema>;

export function CreateXxxForm({ onSuccess }: { onSuccess?: () => void }) {
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
    reset,
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
  });

  const mutation = useMutation({
    mutationFn: (data: FormValues) => xxxService.create(data),
    onSuccess: () => {
      reset();
      onSuccess?.();
    },
  });

  return (
    <form onSubmit={handleSubmit((data) => mutation.mutate(data))} className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="name">Isim</Label>
        <Input id="name" {...register('name')} />
        {errors.name && <p className="text-sm text-destructive">{errors.name.message}</p>}
      </div>

      <div className="space-y-2">
        <Label htmlFor="email">E-posta</Label>
        <Input id="email" type="email" {...register('email')} />
        {errors.email && <p className="text-sm text-destructive">{errors.email.message}</p>}
      </div>

      {mutation.isError && (
        <p className="text-sm text-destructive">
          {mutation.error instanceof Error ? mutation.error.message : 'Bir hata olustu'}
        </p>
      )}

      <Button type="submit" disabled={isSubmitting || mutation.isPending}>
        Olustur
      </Button>
    </form>
  );
}
```

---

## 9. TanStack Query Pattern

```ts
// Query — server data
useQuery({
  queryKey: ['users', { page, role }],   // dependency'leri keyOf parcaisina ekle
  queryFn: () => userService.list({ page, role }),
  staleTime: 5 * 60 * 1000,              // 5 dk fresh — re-fetch yapma
  enabled: !!page,                        // conditional fetch
});

// Mutation — POST/PUT/DELETE
const mutation = useMutation({
  mutationFn: (data: CreateUserRequest) => userService.create(data),
  onSuccess: (newUser) => {
    queryClient.invalidateQueries({ queryKey: ['users'] });          // listeyi yenile
    queryClient.setQueryData(['users', newUser.id], newUser);        // optimistic
  },
  onError: (err) => {
    // toast / log
  },
});
mutation.mutate(formData);
```

**Query key kurallari:**
- Array: `['users']`, `['users', userId]`, `['users', { page, q }]`
- En genelden ozele: `['users']` -> `['users', '123']` -> `['users', '123', 'sessions']`
- Invalidation: `invalidateQueries({ queryKey: ['users'] })` tum varyantlari kapsar

---

## 10. shadcn/ui Kurallari

- `src/components/ui/` icindeki dosyalar shadcn cli ile **uretilmis** — manuel duzenleme **YOK**
- Yeni shadcn componenti: `bunx --bun shadcn@latest add <component>`
- Ozel davranis gerekirse: `src/components/{feature}/` altina **wrapper** yaz, ui'i degistirme

**Sik kullanilan import'lar:**
```ts
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Badge } from '@/components/ui/badge';
```

---

## 11. Tailwind Kurallari (v4)

- `cn()` helper kullan: `import { cn } from '@/lib/utils'`
- Custom CSS yazmadan once Tailwind utility'leri dene
- Renk: theme degiskenleri (`text-foreground`, `bg-background`, `text-destructive`)
- Hardcoded `#hex` veya `rgb()` **YAPMA** — theme degiskenleri kullan
- Responsive: mobile-first (`sm:`, `md:`, `lg:`, `xl:`)

```tsx
<div className={cn(
  'flex items-center gap-2 p-4 rounded-md',
  isActive && 'bg-primary text-primary-foreground',
  variant === 'danger' && 'border-destructive',
)}>
```

### Tailwind 4 — yeni token ekleme
v4'te `tailwind.config` yerine CSS `@theme` blogu kullanilir. `src/index.css`:

```css
@import "tailwindcss";

@theme {
  --color-brand: oklch(0.65 0.2 245);
  --color-brand-foreground: oklch(0.98 0 0);
  --radius: 0.625rem;
}
```

Kullanim: `bg-brand`, `text-brand-foreground`. shadcn'in standart degiskenleri (`--background`, `--foreground`, `--primary`, `--destructive`, `--muted`, `--accent`, `--border`, `--ring`) zaten tanimli — yeni renk gerekirse `@theme` icine ekle, dark mode icin `.dark { --color-brand: ... }` override yaz.

---

## 12. API Client Detaylari (`src/lib/api-client.ts`)

Hazir API instance'lari (her biri Traefik route'una bagli):

```
authApi        → /api/auth
staffApi       → /api/staff
adminStaffApi  → /api/admin-staff
studentApi     → /api/students
catalogApi     → /api/catalog
semesterApi    → /api/semesters
enrollmentApi  → /api/enrollment
attendanceApi  → /api/attendance
gradesApi      → /api/grades
mealApi        → /api/meals
```

**Otomatik davranislar:**
- 401 → `/auth/refresh` cagir → basarili ise request'i tekrarla → basarisiz ise `/auth/login`'e redirect
- CSRF token → cookie'den okunup `X-CSRF-Token` header'ina eklenir
- 408/500/502/503/504 → 2 kez retry
- Trailing `/` → kaldirilir (Gin endpoint match icin)
- `credentials: 'include'` → cookie'ler her zaman gonderilir

**`*ApiSafe` ne zaman kullanilir:**
- Promise.allSettled ile **paralel** istek atarken
- 401 redirect istemiyorsan (admin sistem sayfasi gibi cok servisi cagiran yerlerde)

---

## 13. Mock Mode

```ts
const USE_MOCK = import.meta.env.VITE_USE_MOCK_API === 'true';
import { mockMyGradesResponse } from '@/mock_data/grades';

const data = USE_MOCK ? mockMyGradesResponse : actualData;
```

`.env`'e `VITE_USE_MOCK_API=true` ekleyince backend bagimsiz UI calisir. Yeni mock veri eklerken `src/mock_data/` altina koy.

---

## 14. Kullanici Bildirimi (Toast / Feedback)

> **Mevcut durum**: Projede henuz toast kutuphanesi **kurulu degil**. Mutation `onSuccess`/`onError` icinde kullaniciya geri bildirim, simdilik **inline UI** ile yapiliyor (form altinda `<p className="text-sm text-destructive">...</p>` gibi).

**Mutation feedback pattern'i (toast olmadan):**
```tsx
const [feedback, setFeedback] = useState<{ type: 'success' | 'error'; msg: string } | null>(null);

const mutation = useMutation({
  mutationFn: xxxService.create,
  onSuccess: () => {
    setFeedback({ type: 'success', msg: 'Kayit olusturuldu' });
    queryClient.invalidateQueries({ queryKey: ['xxx'] });
  },
  onError: (err) => {
    setFeedback({
      type: 'error',
      msg: err instanceof Error ? err.message : 'Bir hata olustu',
    });
  },
});
```

Toast eklenirse: `bunx --bun shadcn@latest add sonner` ile **sonner** ekle (shadcn destekli). Eklendikten sonra bu bolum guncellenmeli — kullaniciya sor, dogrudan ekleme (yeni kutuphane karari, bkz. CLAUDE.md Bolum 6).

---

## 15. Erisilebilirlik (a11y) — Minimum

Bu CV/portfolio projesi — full WCAG hedefi yok ama **kirik UI** uretme:

- **Form input**: Her input'un `<Label htmlFor="...">` ile bagli `id`'si olsun (sablon zaten boyle)
- **Icon-only butonlar**: `aria-label` zorunlu — `<Button aria-label="Sil"><Trash /></Button>`
- **Klavye**: shadcn componentleri Radix tabanli, klavye desteklidir; **wrapper'da `onClick` koyup `Pressable` div yapma**
- **Renk kontrasti**: Sadece renkle anlam tasima — error icin `<AlertCircle />` + `text-destructive` ikisi birden
- **Focus visible**: `focus-visible:ring-2 focus-visible:ring-ring` — shadcn varsayilanda var, ezme

---

## 16. Test (Vitest + Testing Library)

```bash
bun run test            # vitest run
bun run test:watch      # watch mode
bun run test:coverage   # coverage raporu
bun run test:ui         # vitest UI
```

**Kurulu**: `vitest`, `@testing-library/react`, `@testing-library/user-event`, `@testing-library/jest-dom`, `jsdom`.

### Ne test ederiz (oncelik sirasi)
1. **Service fonksiyonlari** (`src/lib/services/`) — ky cagrilarini mock'la, request/response sekli dogru mu
2. **Custom hook'lar** (`src/hooks/`) — TanStack Query wrapper'lari `QueryClientProvider` icinde test
3. **Form componentleri** — zod validation + submit akisi
4. Page component test'i opsiyonel (CV projesi)

### Hook test sablonu
```tsx
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useMyXxx } from './useXxx';

function wrapper({ children }: { children: React.ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

test('useMyXxx returns data', async () => {
  const { result } = renderHook(() => useMyXxx(), { wrapper });
  await waitFor(() => expect(result.current.isSuccess).toBe(true));
  expect(result.current.data?.items).toHaveLength(2);
});
```

### Kurallar
- `retry: false` test query client'inda (yoksa hatali test 3x yavaslar)
- Service mock'u: `vi.mock('@/lib/services/xxx-service')` — ky'yi tabandan mock'lama
- `userEvent.setup()` her testte, `fireEvent` yerine `userEvent` tercih
- Snapshot test YAPMA — kirilgan, semantik query (`getByRole`, `getByLabelText`) kullan

---

## 17. Type Generation

```bash
# OpenAPI -> TypeScript types
bun run gen:api-types
# uretir: src/lib/api-types.ts (DOKUNMA)
```

Manuel tip eklemeleri **`src/lib/types.ts`** icinde. `api-types.ts` her regenerate'te ezilir.

---

## 18. Failure Mode Tablosu

| Durum | YAP | YAPMA |
|---|---|---|
| Type error | Tip tanimini duzelt | `as any`, `@ts-ignore` |
| 401 sonsuz dongu | `*ApiSafe` kullan veya auth flow kontrol et | Refresh logic'ini api-client'tan disari tasi |
| TanStack Query stale | `invalidateQueries` cagir | `window.location.reload()` |
| Form submit duplicate | `disabled={isSubmitting}` | Throttle/debounce manuel yaz |
| shadcn component bozuk | `bunx --bun shadcn@latest add <c>` ile yeniden ekle | Manuel `components/ui/` duzenle |
| Build fail | `bun tsc --noEmit` ile lokalde tekrar et, hatayi gor | Hata mesajini gormeden tekrar build |

---

## 19. Yapilmaz Listesi (kisa)

```
Next.js modulleri:        next/navigation, next/link, next/image, next/font
Direktif:                 'use client', 'use server'
Env access:               process.env.X (browser'da undefined)
Routing:                  <a href="/internal">, window.location.href = '/internal'
Server state:             useEffect + fetch + setState
Style:                    inline style={{...}} (cok basit haric), styled-components, CSS modules
Type bypass:              as any, @ts-ignore, @ts-nocheck
Paket yonetici:           npm install, npx <x>, yarn add
HTTP:                     fetch() direkt, axios
Path:                     ../../components/Foo (alias kullan: @/components/Foo)
```
