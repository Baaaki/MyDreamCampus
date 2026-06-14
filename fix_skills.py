path = "/home/nautilus/Desktop/Playground/mydreamcampus/frontend/skills.md"

new_content = """# Frontend — React + Vite (AI Talimati)

React 19 + Vite + react-router v7 + TanStack Query + ky + shadcn/ui + Tailwind 4. `frontend/src/**` icinde calisirken bu dosya zorunlu okumadir.

> **Onemli:** Bu Next.js DEGIL. Vite. `'use client'` yok, `next/*` yok.

---

## 1. Sert Kurallar (asla ihlal etme)

- **Paket yoneticisi**: `bun` — `npm`, `npx`, `yarn` YAPMA. Tip kontrolu icin `bun tsc --noEmit`.
- **Routing**: `react-router` v7 — `next/navigation`, `next/link` import etme.
- **HTTP**: `ky` (`src/lib/api-client.ts`) — `fetch` direkt veya `axios` kullanma.
- **Server state**: `@tanstack/react-query` — `useEffect` icinde fetch YAPMA.
- **'use client' direktifi YOK** — Vite SSR yapmiyor.
- **Env var prefix**: `VITE_` — `process.env` YAPMA, `import.meta.env.VITE_X` kullan.
- **Alias**: `@/` -> `src/` — relative path (`../../components/...`) YAPMA.
- **CSRF**: `api-client.ts` cookie'den `X-CSRF-Token` ekliyor — manuel ekleme.

---

## 2. Dosya Yapisi (sabit)

`pages/` (Sayfalar), `components/ui/` (shadcn - DUZENLEME), `lib/services/` (API cagrilar), `routes.tsx` (Route tanimlari).

---

## 3. Yeni Sayfa Workflow

1. Type tanimi (`src/lib/types.ts`)
2. Service (`src/lib/services/`)
3. Page bilesen (`src/pages/`)
4. Route ekle (`src/routes.tsx`)
5. Test (`bun run dev`, `bun tsc --noEmit`)

---

## 4. State Karar Matrisi (zorunlu uy)

- **Server data**: TanStack Query (`useQuery`, `useMutation`)
- **Form**: `react-hook-form` + `zod`
- **URL/Filtre**: `useSearchParams` (react-router)
- **Local UI**: `useState`
- **Side effect**: `useEffect` (Fetch icin DEGIL)

---

## 5. Routing Hizli Referans

`react-router` kullanilir. `<Link to="...">`, `useNavigate()`, `useParams()`, `useSearchParams()`.
**YAPMA:** `<a href="...">` veya `window.location.href`.

---

## 6. Service & API Pattern

- `import { xxxApi } from '@/lib/api-client'`
- `ky` kurallari: `xxxApi.post('', { json: data }).json<T>()`.
- 401 ve CSRF `api-client.ts` tarafindan yonetilir.

---

## 7. Component & Form Pattern

- **Loading/Error**: TanStack Query `isLoading`, `isError` kullan. Loading icin `Loader2`, Error icin `AlertCircle` + Turkce mesaj.
- **Mutation Sonrasi**: `queryClient.invalidateQueries` cagir.
- **Formlar**: `useForm` (react-hook-form) + `zodResolver`.

---

## 8. shadcn/ui & Tailwind (v4) Kurallari

- `bunx --bun shadcn@latest add <component>` ile ekle, `components/ui/` icini manuel duzenleme.
- Tailwind class'larini birlestirmek icin `cn()` kullan.
- Hardcoded renk yerine theme token'larini kullan (bg-background, text-destructive vs). Tailwind v4 `@theme` (index.css) kullanir.

---

## 9. Test (Vitest)

`bun run test`. Service fonksiyonlari ve hook'lari test edilir. Snapshot veya kirilgan test yapma.

---

## 10. Failure Mode Tablosu

| Durum | YAP | YAPMA |
|---|---|---|
| Type error | Tip tanimini duzelt | `as any`, `@ts-ignore` |
| Form duplicate submit | `disabled={isSubmitting}` | Throttle manuel yaz |
| shadcn bozuk | `bunx shadcn add <c>` ile yenile | Manuel duzenle |
"""

with open(path, "w") as f:
    f.write(new_content)
print("skills.md shortened")
