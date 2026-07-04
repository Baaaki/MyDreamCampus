# Mobile — React Native + Expo (AI Talimati)

React Native 0.81 + Expo 54 + Expo Router v6 + TanStack Query + axios. `mobile/app/**`, `mobile/services/**`, `mobile/hooks/**` icinde calisirken bu dosya zorunlu okumadir.

> **Onemli:** Bu web frontend DEGIL. React Router YOK, ky YOK, shadcn YOK.

---

## 1. Sert Kurallar (asla ihlal etme)

- **Paket yoneticisi**: `npm` — `bun` YAPMA (eskiden vardi, kaldirildi). Tip kontrolu icin `npx tsc --noEmit`.
- **Routing**: `expo-router` v6 file-based — `react-router`, `next/navigation` YAPMA.
- **HTTP**: `axios` (`mobile/services/api.ts`) — `fetch` direkt veya `ky` YAPMA.
- **Server state**: `@tanstack/react-query` — manuel `useEffect`+`useState` ile fetch YAPMA.
- **Token storage**: `expo-secure-store` — `AsyncStorage`, `localStorage` YAPMA (token icin).
- **Env var**: `EXPO_PUBLIC_*` prefix — `process.env.EXPO_PUBLIC_X`.
- **Web/native ayrimi**: `localStorage`, `document`, `window` ortak kodda YAPMA — web target'a ozel davranis gerekirse `Platform.OS === 'web'` guard ile koru veya `.web.ts` suffix dosya olarak ayir. Native build'de bu API'lar yok, runtime crash olur.
- **Style**: **NativeWind v4** (`className="..."`) + `components/ui/` primitifleri (react-native-reusables pattern: cva + cn). Tema renkleri `global.css` CSS degiskenlerinden gelir (`bg-primary`, `text-foreground` vs.) — hardcoded hex YAPMA. `className` ile ifade edilemeyen degerler (animasyon transform'lari, shadow renkleri, yuzde genislik) icin `style={{...}}` serbest; JS tarafinda renk gerekirse `lib/theme.ts` `COLORS`. `react-native-paper` KALDIRILDI — geri ekleme.
- **Tema (dark mode)**: `contexts/ThemeContext` (`useTheme()` -> `isDark`, `toggleTheme`) NativeWind `colorScheme` uzerine sarilidir. Renk degistirmek icin `global.css` (`:root` / `.dark:root`) + `lib/theme.ts` ikisini birden guncelle.

---

## 2. Dosya Yapisi (sabit)

Uygulama **sadece ogrenci** icindir — personel/ogretmen/admin akisi YOK (web frontend'de var).

```
mobile/
├── app/                       # Expo Router file-based routing
│   ├── _layout.tsx            # Root layout (provider'lar + AuthGuard + Stack)
│   ├── (auth)/                # login
│   ├── (tabs)/                # index (ana sayfa), scan (QR yoklama), profile
│   ├── change-password.tsx    # zorunlu sifre degisimi (modal)
│   ├── +not-found.tsx         # 404
│   └── +html.tsx              # web HTML wrapper (RN web)
├── components/ui/             # tasarim sistemi primitifleri (text, button, card, input, badge)
├── contexts/                  # AuthContext, ThemeContext
├── hooks/                     # useAuth, useAttendance, useEnrollment, useCatalog, useHaptic
├── services/                  # API service'leri
│   ├── api.ts                 # axios instance + 401 refresh interceptor
│   ├── authService.ts         # login, logout, refresh
│   └── {feature}Service.ts
├── lib/                       # utils (cn), theme (COLORS/NAV_THEME), qr-payload, password-policy
├── constants/                 # schedule.ts (TIME_SLOTS, DAYS_OF_WEEK)
├── types/                     # TypeScript tipleri
├── global.css                 # tema CSS degiskenleri (:root / .dark:root)
├── tailwind.config.js         # NativeWind preset + token eslemeleri
└── assets/                    # gorseller, fontlar (SpaceMono = data/eyebrow fontu)
```

**Route grup parantezleri** (Expo Router):
- `(auth)`, `(tabs)` → URL'de **gorunmez**, sadece organize amacli

---

## 3. Yeni Ekran Workflow

```
1. Type tanimi:    types/{feature}.types.ts
2. Service:        services/{feature}Service.ts (Bolum 6 sablonu)
3. Hook:           hooks/use{Feature}.ts (Bolum 9 sablonu)
4. Ekran:          app/{group}/{feature}.tsx (Bolum 7 sablonu)
5. Layout (varsa): app/{group}/_layout.tsx (Stack/Tabs ekle)
6. Tip kontrol:    npx tsc --noEmit
7. Test:           npm test
8. Cihazda dene:   npm start -> a (Android) / i (iOS)
9. Commit:         feat(mobile): add {feature} screen
```

---

## 4. Routing — Expo Router

```tsx
import { useRouter, useLocalSearchParams, Link, Stack, Tabs } from 'expo-router';

// Programmatic
const router = useRouter();
router.push('/screens/detail/123');
router.replace('/(tabs)/home');                  // history replace
router.back();
router.dismiss();                                 // modal kapat

// Link
<Link href="/screens/detail/123">Detay</Link>    // ✅ href (to DEGIL)

// URL parametre — dosya: app/screens/detail/[id].tsx
const { id } = useLocalSearchParams<{ id: string }>();

// Query — /list?page=2
const { page } = useLocalSearchParams<{ page?: string }>();
```

**Kurallar:**
- Route adi = dosya adi (`app/profile.tsx` → `/profile`)
- Dynamic: `[id].tsx` → `/123` ile match
- Layout: `_layout.tsx` (her klasorde optional)
- Modal: `presentation: 'modal'` Stack option

**YAPMA:**
- `useNavigate()` (react-router'a ait)
- `useParams()` (react-router'a ait — Expo Router `useLocalSearchParams`)
- `<Link to="...">` — `href` kullan

---

## 5. State Karar Matrisi

| Veri turu | Kullan |
|---|---|
| Server data | **TanStack Query** (`useQuery`, `useMutation`) |
| Form state | `react-hook-form` + `zod` |
| URL params | `useLocalSearchParams` (Expo Router) |
| Modal/sheet acik-kapali | `useState` |
| Auth durumu, tema | Context (`AuthContext`, `ThemeContext`) |
| Hesaplanmis | `useMemo` |
| Token (persist) | `expo-secure-store` |
| User profili (persist, az kritik) | `expo-secure-store` veya `AsyncStorage` |

**YAPMA:**
- Token'i `AsyncStorage`'a koy (encrypt edilmiyor)
- Server data'yi `useState`+`useEffect` ile cek
- Redux/Zustand ekle (proje minimum)

---

## 6. Service Sablonu (`mobile/services/`)

```ts
// mobile/services/xxxService.ts
import api from './api';
import type {
  CreateXxxRequest,
  XxxResponse,
  XxxListResponse,
} from '@/types/xxx.types';

export const xxxService = {
  async list(params?: { page?: number; q?: string }): Promise<XxxListResponse> {
    const response = await api.get<XxxListResponse>('/xxx', { params });
    return response.data;
  },

  async getById(id: string): Promise<XxxResponse> {
    const response = await api.get<XxxResponse>(`/xxx/${id}`);
    return response.data;
  },

  async create(data: CreateXxxRequest): Promise<XxxResponse> {
    const response = await api.post<XxxResponse>('/xxx', data);
    return response.data;
  },

  async update(id: string, data: Partial<CreateXxxRequest>): Promise<XxxResponse> {
    const response = await api.put<XxxResponse>(`/xxx/${id}`, data);
    return response.data;
  },

  async remove(id: string): Promise<void> {
    await api.delete(`/xxx/${id}`);
  },
};

export default xxxService;
```

**axios kurallari:**
- `api` instance'i her zaman kullan (`api.get`, `api.post`) — yeni instance kurma
- Path kok ile (`/xxx`) — base URL `services/api.ts` icinde tanimli (`/api` ekli)
- 401 → `services/api.ts` interceptor otomatik refresh + retry
- Token → SecureStore'dan otomatik request header'ina ekleniyor
- Response data: `response.data` (axios `.data` doner, fetch gibi `.json()` DEGIL)

---

## 7. Ekran Component Sablonu

```tsx
// app/(tabs)/xxx.tsx
import { ActivityIndicator, FlatList, Pressable, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import Animated, { FadeInDown } from 'react-native-reanimated';

import { Button, Card, Text } from '@/components/ui';
import { useMyXxx } from '@/hooks/useXxx';

export default function XxxScreen() {
  const router = useRouter();
  const { data, isLoading, isError, error, refetch } = useMyXxx();

  if (isLoading) {
    return (
      <SafeAreaView className="flex-1 items-center justify-center bg-background" edges={['top']}>
        <ActivityIndicator size="large" />
      </SafeAreaView>
    );
  }

  if (isError) {
    return (
      <SafeAreaView
        className="flex-1 items-center justify-center gap-4 bg-background px-8"
        edges={['top']}
      >
        <Text className="text-center text-muted-foreground">
          {error instanceof Error ? error.message : 'Bir hata olustu'}
        </Text>
        <Button variant="outline" onPress={() => refetch()} accessibilityLabel="Tekrar dene">
          <Text>Tekrar Dene</Text>
        </Button>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView className="flex-1 bg-background" edges={['top']}>
      <FlatList
        data={data?.items ?? []}
        keyExtractor={(item) => item.id}
        contentContainerClassName="gap-3 p-5"
        renderItem={({ item, index }) => (
          <Animated.View entering={FadeInDown.delay(index * 60)}>
            <Pressable
              onPress={() => router.push(`/xxx/${item.id}`)}
              accessibilityRole="button"
              className="active:opacity-70"
            >
              <Card className="p-4">
                <Text className="font-semibold">{item.name}</Text>
              </Card>
            </Pressable>
          </Animated.View>
        )}
        ListEmptyComponent={
          <Text className="p-8 text-center text-muted-foreground">Henuz kayit yok</Text>
        }
        onRefresh={refetch}
        refreshing={isLoading}
      />
    </SafeAreaView>
  );
}
```

**Ekran kurallari:**
- Default export — Expo Router otomatik route bulur
- **`SafeAreaView` (`react-native-safe-area-context`)** — root container, `className="flex-1 bg-background"` + `edges={['top']}` (alt tab bar zaten safe). **Pure `react-native`'in `SafeAreaView`'unu kullanma** (Android'de calismiyor).
- `FlatList` (uzun liste) > `ScrollView` + `.map()`
- `keyExtractor` zorunlu (`(item) => item.id`)
- Loading: `ActivityIndicator`; empty state: `ListEmptyComponent` veya manuel kontrol
- Buton/kart/input/badge icin once `components/ui/` primitiflerine bak, yoksa ekle — ekranda tek seferlik stil uydurma
- Pressable feedback: `className="active:opacity-70"` (NativeWind active: state)
- Mikro-animasyon: `react-native-reanimated` `entering={FadeInDown...}` — abartma, liste/section girisleri yeterli
- A11y minimum: `accessibilityRole="button"`, ikon-only butonlarda `accessibilityLabel`

---

## 8. Form Sablonu (react-hook-form + zod)

> Not: Sablondaki `StyleSheet` stilleri tarihseldir — yeni formlarda input/buton icin
> `components/ui` (`Input`, `Button`, `Text`) + `className` kullan; form **logic'i**
> (Controller, zodResolver, mutation) aynen gecerli.

```tsx
import {
  View, Text, TextInput, Pressable, StyleSheet, Alert,
  KeyboardAvoidingView, ScrollView, Platform, Keyboard, TouchableWithoutFeedback,
} from 'react-native';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useMutation } from '@tanstack/react-query';
import { xxxService } from '@/services/xxxService';

const schema = z.object({
  name: z.string().min(2, 'En az 2 karakter').max(100, 'En fazla 100 karakter'),
  email: z.string().email('Gecerli bir e-posta giriniz'),
});

type FormValues = z.infer<typeof schema>;

export function CreateXxxForm({ onSuccess }: { onSuccess?: () => void }) {
  const { control, handleSubmit, formState: { errors }, reset } = useForm<FormValues>({
    resolver: zodResolver(schema),
  });

  const mutation = useMutation({
    mutationFn: (data: FormValues) => xxxService.create(data),
    onSuccess: () => {
      reset();
      onSuccess?.();
    },
    onError: (err) => {
      Alert.alert('Hata', err instanceof Error ? err.message : 'Bir hata olustu');
    },
  });

  return (
    <KeyboardAvoidingView
      style={{ flex: 1 }}
      behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
    >
      <TouchableWithoutFeedback onPress={Keyboard.dismiss}>
        <ScrollView contentContainerStyle={styles.form} keyboardShouldPersistTaps="handled">
          <Controller
            control={control}
            name="name"
            render={({ field: { onChange, value } }) => (
              <TextInput
                style={styles.input}
                placeholder="Isim"
                value={value}
                onChangeText={onChange}
                returnKeyType="next"
                accessibilityLabel="Isim"
              />
            )}
          />
          {errors.name && <Text style={styles.error}>{errors.name.message}</Text>}

          <Controller
            control={control}
            name="email"
            render={({ field: { onChange, value } }) => (
              <TextInput
                style={styles.input}
                placeholder="E-posta"
                keyboardType="email-address"
                autoCapitalize="none"
                autoComplete="email"
                value={value}
                onChangeText={onChange}
                returnKeyType="done"
                accessibilityLabel="E-posta"
              />
            )}
          />
          {errors.email && <Text style={styles.error}>{errors.email.message}</Text>}

          <Pressable
            style={[styles.button, mutation.isPending && styles.buttonDisabled]}
            onPress={handleSubmit((data) => mutation.mutate(data))}
            disabled={mutation.isPending}
            accessibilityRole="button"
          >
            <Text style={styles.buttonText}>
              {mutation.isPending ? 'Gonderiliyor...' : 'Olustur'}
            </Text>
          </Pressable>
        </ScrollView>
      </TouchableWithoutFeedback>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  form: { gap: 12, padding: 16 },
  input: { borderWidth: 1, borderColor: '#ddd', borderRadius: 8, padding: 12, fontSize: 16 },
  error: { color: 'red', fontSize: 12 },
  button: { padding: 14, backgroundColor: '#007aff', borderRadius: 8, alignItems: 'center' },
  buttonDisabled: { opacity: 0.5 },
  buttonText: { color: '#fff', fontWeight: '600', fontSize: 16 },
});
```

**Kurallar:**
- React Native input'lari uncontrolled olamaz — `Controller` ile sar
- `KeyboardAvoidingView` ile sar — iOS'ta klavye input'u kapatir, `behavior="padding"`. Android'de `"height"` veya `null` (genelde `android:windowSoftInputMode="adjustResize"` zaten halleder ama ek olarak zarar vermez)
- `keyboardShouldPersistTaps="handled"` — input'tan sonra butona dokunmak icin
- `keyboardType="email-address"` / `numeric` / `phone-pad` — UX
- `autoCapitalize="none"` — email/username icin
- `autoComplete="email"` / `password` / `password-new` — iOS keychain, Android autofill
- `returnKeyType="next"` — next field; son input'ta `"done"`
- Hata mesajlari: `Alert.alert` veya inline `Text`

---

## 9. TanStack Query Hook Sablonu

```ts
// mobile/hooks/useXxx.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { xxxService } from '@/services/xxxService';

export const useMyXxx = () =>
  useQuery({
    queryKey: ['my-xxx'],
    queryFn: () => xxxService.list(),
    staleTime: 5 * 60 * 1000,
  });

export const useXxxById = (id: string, enabled = true) =>
  useQuery({
    queryKey: ['xxx', id],
    queryFn: () => xxxService.getById(id),
    enabled: !!id && enabled,
    staleTime: 5 * 60 * 1000,
  });

export const useCreateXxx = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Parameters<typeof xxxService.create>[0]) => xxxService.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['my-xxx'] });
    },
  });
};
```

---

## 10. Auth & SecureStore

```ts
import * as SecureStore from 'expo-secure-store';

const TOKEN_KEY = 'jwt_token';
const REFRESH_KEY = 'refresh_token';
const USER_KEY = 'user_data';

await SecureStore.setItemAsync(TOKEN_KEY, accessToken);
const token = await SecureStore.getItemAsync(TOKEN_KEY);
await SecureStore.deleteItemAsync(TOKEN_KEY);
```

**Kural:**
- Token'lar SecureStore (encrypted) — `AsyncStorage` YAPMA
- User profili optional SecureStore (hassas alanlar varsa zorunlu)
- Logout temizligi: 3 anahtar da silinmeli (`TOKEN`, `REFRESH`, `USER`)

`services/api.ts` 401 interceptor zaten:
- Refresh token'i SecureStore'dan okur
- `/auth/refresh` cagirir
- Basarili → yeni token kaydet, original request'i tekrarla
- Basarisiz → 3 anahtari sil, `onUnauthorized` callback (ekran `/login`'e yonlendirir)

---

## 11. Platform Farklari

```ts
import { Platform } from 'react-native';

if (Platform.OS === 'ios') { /* ... */ }
if (Platform.OS === 'android') { /* ... */ }
if (Platform.OS === 'web') { /* ... */ }
```

**Dosya suffix'leri:**
- `useColorScheme.web.ts` → sadece web build'de
- `useColorScheme.ts` → ios/android default

API base URL ornek (`services/api.ts`):
```ts
const getBaseURL = () => {
  const envUrl = process.env.EXPO_PUBLIC_API_URL;
  if (envUrl) return envUrl;
  if (Platform.OS === 'android') return 'http://10.0.2.2/api';  // Android emulator
  return 'http://localhost/api';                                  // iOS simulator
};
```

**Gercek cihaz (LAN) testi:**
Simulator/emulator yerine telefonda Expo Go kullaniliyorsa `localhost`/`10.0.2.2` calismaz — telefon ev/ofis WiFi'sinden Mac/PC'ye ulasamaz. Cozum:
1. Mac/PC'nin LAN IP'sini bul: `ipconfig getifaddr en0` (Mac) / `ip addr show` (Linux) / `ipconfig` (Windows)
2. `.env`: `EXPO_PUBLIC_API_URL=http://192.168.1.X/api`
3. Telefon ayni WiFi'de olmali, firewall 80 portunu izin vermeli
4. `npm start` -> QR kodunu Expo Go ile tara

---

## 12. Henuz Eklenmemis (kullanici onayi gerekir)

Bu ozellikler **henuz kurulu degil**. Eklenmesi yeni dependency + native config gerektirir — kullaniciya **sor** sonra ekle (CLAUDE.md Bolum 6).

| Ozellik | Paket | Ne icin |
|---|---|---|
| Push notifications | `expo-notifications` | Devamsizlik bildirimi, not aciklanmasi vs. |
| Deep linking | `expo-router` (zaten var) + `app.json` `scheme` | Universal links, email'den uygulama acma |
| Image optimization | `expo-image` | `<Image>` yerine cache + placeholder |
| EAS Build | `eas-cli` (npx ile) | Production .ipa/.apk. Expo Go yetmez ise |
| OTA update | `expo-updates` | Native build atmadan JS bundle deploy |

EAS build hizli rehber:
```bash
npx eas-cli build --profile preview --platform android
# eas.json'da profile tanimli olmali; ilk seferde npx eas-cli build:configure
```

---

## 13. Type Generation

```bash
# Backend OpenAPI -> TypeScript
npm run gen:api-types
# uretir: types/api-types.ts (DOKUNMA)
```

Manuel tipler `types/{feature}.types.ts` icinde.

---

## 14. Test (Jest)

```bash
npm test              # tum testler
npm run test:watch    # watch mode
npm run test:coverage # kapsam raporu
```

- Setup: `jest.setup.js`
- Mock'lar: `__mocks__/` (expo-secure-store, axios vs.)
- Test dosyasi: `*.test.ts` (component icin `*.test.tsx`)

**Kural:** Service ve hook'lar test edilir. Ekran (component) test'i opsiyonel (CV projesi, minimum kapsam).

---

## 15. Failure Mode Tablosu

| Durum | YAP | YAPMA |
|---|---|---|
| Type error | Tipi duzelt | `as any`, `@ts-ignore` |
| Metro cache bozuk | `npx expo start -c` (cache temizle) | `node_modules` sil |
| Build fail | Expo Go yeterli mi kontrol (native modul varsa Dev Client gerekir) | EAS build calistirilirken log'lar olmadan hata ayiklamaya kalkma |
| 401 sonsuz dongu | `services/api.ts` _retried flag kontrol | Interceptor'i bypass et |
| iOS simulator API'ye ulasamiyor | Mac IP'sini `EXPO_PUBLIC_API_URL`'e yaz | `localhost` Android'de calismaz |
| Android emulator API'ye ulasamiyor | `http://10.0.2.2/api` kullan | `localhost` direkt |
| SecureStore null | iOS keychain reset olmus olabilir, login'e yonlendir | Crash'le, hata yutma |

---

## 16. Yapilmaz Listesi (kisa)

```
Web modulleri:        next/*, react-router, ky
Storage:              localStorage (token icin AsyncStorage bile YAPMA)
Style:                styled-components, react-native-paper (kaldirildi), hardcoded hex renk (tema degiskeni varken)
Routing API:          useNavigate, useParams (react-router)
HTTP:                 fetch direkt, axios.create yeni instance
Token:                JWT'yi AsyncStorage'a koymak (SecureStore zorunlu)
Type bypass:          as any, @ts-ignore, @ts-nocheck
Liste:                ScrollView + .map() (uzun liste icin — FlatList kullan)
Paket:                bun (kaldirildi), yarn
Path:                 ../../components/Foo (alias kullan: @/components/Foo)
Env:                  process.env.X (prefix yoksa undefined — EXPO_PUBLIC_ kullan)
```
